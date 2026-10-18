// Package elfinfo reads ELF metadata without executing a binary.
package elfinfo

import (
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrNotELF      = errors.New("not an ELF executable")
	ErrUnsupported = errors.New("unsupported ELF architecture")
)

// Binary describes the ELF properties needed to select a glibc loader.
type Binary struct {
	Path        string
	Machine     elf.Machine
	Class       elf.Class
	Type        elf.Type
	Interpreter string
}

// Inspect reads the ELF header and PT_INTERP; it never executes the file.
func Inspect(path string) (Binary, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Binary{}, err
	}
	file, err := os.Open(absolute)
	if err != nil {
		return Binary{}, err
	}
	defer file.Close()

	parsed, err := elf.NewFile(file)
	if err != nil {
		return Binary{}, fmt.Errorf("%w: %s: %v", ErrNotELF, absolute, err)
	}
	defer parsed.Close()
	if parsed.Type != elf.ET_EXEC && parsed.Type != elf.ET_DYN {
		return Binary{}, fmt.Errorf("%w: unsupported ELF type %s", ErrNotELF, parsed.Type)
	}

	info := Binary{Path: absolute, Machine: parsed.Machine, Class: parsed.Class, Type: parsed.Type}
	for _, program := range parsed.Progs {
		if program.Type != elf.PT_INTERP {
			continue
		}
		if info.Interpreter != "" {
			return Binary{}, errors.New("ELF has multiple PT_INTERP headers")
		}
		if program.Filesz == 0 || program.Filesz > 4096 {
			return Binary{}, errors.New("invalid ELF PT_INTERP length")
		}
		data, err := io.ReadAll(program.Open())
		if err != nil {
			return Binary{}, fmt.Errorf("read ELF interpreter: %w", err)
		}
		if len(data) == 0 || data[len(data)-1] != 0 || strings.ContainsRune(string(data[:len(data)-1]), 0) {
			return Binary{}, errors.New("invalid ELF PT_INTERP string")
		}
		info.Interpreter = string(data[:len(data)-1])
		if !filepath.IsAbs(info.Interpreter) {
			return Binary{}, errors.New("ELF interpreter is not absolute")
		}
	}
	return info, nil
}

// Glibc reports whether the ELF declares a recognized glibc interpreter.
// It does not attempt to classify static binaries or verify ABI compatibility.
func (b Binary) Glibc() bool {
	switch b.Machine {
	case elf.EM_X86_64:
		return b.Class == elf.ELFCLASS64 && (b.Interpreter == "/lib64/ld-linux-x86-64.so.2" || b.Interpreter == "/lib/ld-linux-x86-64.so.2")
	case elf.EM_AARCH64:
		return b.Class == elf.ELFCLASS64 && b.Interpreter == "/lib/ld-linux-aarch64.so.1"
	default:
		return false
	}
}

func Architecture(machine elf.Machine) (string, error) {
	switch machine {
	case elf.EM_X86_64:
		return "x86_64", nil
	case elf.EM_AARCH64:
		return "aarch64", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupported, machine)
	}
}
