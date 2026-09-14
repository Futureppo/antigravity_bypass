package main

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"sort"

	"golang.org/x/arch/x86/x86asm"
)

const toolCountError = "number of tools exceeds %d"

type binarySection struct {
	offset, size int
	address      uint64
}

type binaryLayout struct {
	text     binarySection
	sections []binarySection
}

func readBinaryLayout(data []byte) (binaryLayout, error) {
	var layout binaryLayout
	add := func(name string, offset, size, address uint64) error {
		if offset > uint64(len(data)) || size > uint64(len(data))-offset {
			return fmt.Errorf("无效的 %s 节区范围", name)
		}
		s := binarySection{int(offset), int(size), address}
		layout.sections = append(layout.sections, s)
		if name == ".text" || name == "__text" {
			layout.text = s
		}
		return nil
	}
	switch {
	case bytes.HasPrefix(data, []byte("MZ")):
		f, err := pe.NewFile(bytes.NewReader(data))
		if err != nil {
			return layout, err
		}
		defer f.Close()
		if f.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
			return layout, fmt.Errorf("语言服务器不是 x64；暂不支持此架构的二进制修补")
		}
		header, ok := f.OptionalHeader.(*pe.OptionalHeader64)
		if !ok {
			return layout, fmt.Errorf("无效的 PE64 头")
		}
		for _, s := range f.Sections {
			if s.Size == 0 {
				continue
			}
			if err := add(s.Name, uint64(s.Offset), uint64(s.Size), header.ImageBase+uint64(s.VirtualAddress)); err != nil {
				return layout, err
			}
		}
	case bytes.HasPrefix(data, []byte{0x7f, 'E', 'L', 'F'}):
		f, err := elf.NewFile(bytes.NewReader(data))
		if err != nil {
			return layout, err
		}
		defer f.Close()
		if f.Machine != elf.EM_X86_64 || f.Class != elf.ELFCLASS64 || f.Data != elf.ELFDATA2LSB {
			return layout, fmt.Errorf("语言服务器不是小端 x64 ELF；暂不支持此架构的二进制修补")
		}
		for _, s := range f.Sections {
			if s.Type == elf.SHT_NOBITS || s.Flags&elf.SHF_ALLOC == 0 || s.Size == 0 {
				continue
			}
			if err := add(s.Name, s.Offset, s.Size, s.Addr); err != nil {
				return layout, err
			}
		}
	default:
		f, err := macho.NewFile(bytes.NewReader(data))
		if err != nil {
			return layout, fmt.Errorf("不支持或损坏的二进制格式: %w", err)
		}
		defer f.Close()
		if f.Cpu != macho.CpuAmd64 {
			return layout, fmt.Errorf("语言服务器不是 x64 Mach-O；暂不支持此架构的二进制修补")
		}
		for _, s := range f.Sections {
			if s.Flags&0xff == 1 || s.Flags&0xff == 0xc || s.Flags&0xff == 0x12 || s.Size == 0 {
				continue
			} // zero-fill
			if err := add(s.Name, uint64(s.Offset), s.Size, s.Addr); err != nil {
				return layout, err
			}
		}
	}
	if layout.text.size == 0 {
		return layout, fmt.Errorf("未找到有效代码节区；拒绝扫描整个文件")
	}
	return layout, nil
}

type limitSite struct {
	offset   int
	original uint32
}

// errorReferences requires both the address of the actual Go format string and
// its length argument. Identical bytes in data sections are never executable candidates.
func errorReferences(data []byte, layout binaryLayout) []int {
	addresses := make(map[uint64]bool)
	for _, s := range layout.sections {
		section := data[s.offset : s.offset+s.size]
		for pos := 0; pos < len(section); {
			i := bytes.Index(section[pos:], []byte(toolCountError))
			if i < 0 {
				break
			}
			i += pos
			addresses[s.address+uint64(i)] = true
			pos = i + len(toolCountError)
		}
	}
	t := layout.text
	code := data[t.offset : t.offset+t.size]
	var refs []int
	for i := 0; i+12 <= len(code); i++ {
		if !bytes.Equal(code[i:i+3], []byte{0x48, 0x8d, 0x1d}) || code[i+7] != 0xb9 || binary.LittleEndian.Uint32(code[i+8:]) != uint32(len(toolCountError)) {
			continue
		}
		address := int64(t.address) + int64(i) + 7 + int64(int32(binary.LittleEndian.Uint32(code[i+3:])))
		if address >= 0 && addresses[uint64(address)] {
			refs = append(refs, i)
		}
	}
	return refs
}

// Verify that the conditional branch leads directly into the error's argument
// setup, without crossing another branch, call or return on the way to the LEA.
func isErrorBlock(code []byte, start, ref int) bool {
	if start < 0 || start > ref || ref-start > 128 {
		return false
	}
	for pos := start; pos < ref; {
		inst, err := x86asm.Decode(code[pos:ref], 64)
		if err != nil || inst.Len == 0 || inst.Op == 0 {
			return false
		}
		switch inst.Op {
		case x86asm.LEA, x86asm.MOV, x86asm.XOR, x86asm.NOP:
		default:
			return false
		}
		pos += inst.Len
		if pos > ref {
			return false
		}
	}
	return true
}

func findLimitSites(data []byte, layout binaryLayout) ([]limitSite, error) {
	refs := errorReferences(data, layout)
	if len(refs) != 2 {
		return nil, fmt.Errorf("需要确认两处工具数量报错引用，实际找到 %d 处；不使用旧版全局指令替换", len(refs))
	}
	t := layout.text
	code := data[t.offset : t.offset+t.size]
	byRef := make(map[int][]limitSite)
	for i := 0; i+13 <= len(code); i++ {
		// CMP r64, imm32. 128/512 require imm32, unlike the obsolete CMP ...,100 signature.
		if (code[i] != 0x48 && code[i] != 0x49) || code[i+1] != 0x81 || code[i+2] < 0xf8 {
			continue
		}
		value := binary.LittleEndian.Uint32(code[i+3:])
		if value != 128 && value != 512 && value != 114514 && value != toolLimit {
			continue
		}
		branch, end, target, condition := i+7, 0, 0, byte(0)
		if code[branch] == 0x7e || code[branch] == 0x7f {
			condition = code[branch] & 0xf
			end = branch + 2
			target = end + int(int8(code[branch+1]))
		} else if code[branch] == 0x0f && (code[branch+1] == 0x8e || code[branch+1] == 0x8f) {
			condition = code[branch+1] & 0xf
			end = branch + 6
			target = end + int(int32(binary.LittleEndian.Uint32(code[branch+2:])))
		} else {
			continue
		}
		if target <= end || target >= len(code) {
			continue
		}
		for _, ref := range refs {
			start := target       // JG enters the error block.
			if condition == 0xe { // JLE skips the returning error block.
				if target <= ref+12 || target-end > 256 || code[target-1] != 0xc3 {
					continue
				}
				start = end
			}
			if isErrorBlock(code, start, ref) {
				byRef[ref] = append(byRef[ref], limitSite{t.offset + i + 3, value})
			}
		}
	}
	var sites []limitSite
	for _, ref := range refs {
		if len(byRef[ref]) != 1 {
			return nil, fmt.Errorf("工具报错引用 0x%x 对应 %d 个检查，无法安全确定目标", t.offset+ref, len(byRef[ref]))
		}
		sites = append(sites, byRef[ref][0])
	}
	sort.Slice(sites, func(i, j int) bool { return sites[i].offset < sites[j].offset })
	for i := 1; i < len(sites); i++ {
		if sites[i].offset == sites[i-1].offset {
			return nil, fmt.Errorf("多个报错引用共享同一检查，未修改")
		}
	}
	return sites, nil
}

func planLanguageServer(data []byte) ([]byte, []string, error) {
	layout, err := readBinaryLayout(data)
	if err != nil {
		return nil, nil, err
	}
	sites, err := findLimitSites(data, layout)
	if err != nil {
		return nil, nil, err
	}
	updated := bytes.Clone(data)
	var messages []string
	for _, site := range sites {
		if site.original == toolLimit {
			messages = append(messages, fmt.Sprintf("后端 0x%08x: 已修补", site.offset))
		} else {
			binary.LittleEndian.PutUint32(updated[site.offset:], toolLimit)
			messages = append(messages, fmt.Sprintf("后端 0x%08x: %d -> %d", site.offset, site.original, toolLimit))
		}
	}
	return updated, messages, nil
}
