// Package launcher prepares a non-privileged glibc process invocation.
package launcher

import (
	"debug/elf"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/angyedz/veyra-glenv/internal/elfinfo"
	glruntime "github.com/angyedz/veyra-glenv/internal/runtime"
)

var ErrNotGlibc = errors.New("binary does not use a supported glibc interpreter")

// Plan is a complete loader invocation. No process is started by PlanFor.
type Plan struct {
	Binary  elfinfo.Binary
	Runtime glruntime.Runtime
	Loader  string
	Args    []string
	Preload string // optional original glenv Wayland filter
}

// PlanFor accepts an executable path or an executable name from PATH.
// It refuses musl, static and unrecognized ELF programs rather than running
// them under an unrelated dynamic loader.
func PlanFor(root, program string, args []string) (Plan, error) {
	if program == "" {
		return Plan{}, errors.New("missing program")
	}
	path := program
	if !strings.ContainsRune(path, filepath.Separator) {
		resolved, err := exec.LookPath(path)
		if err != nil {
			return Plan{}, err
		}
		path = resolved
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Plan{}, err
	}
	stat, err := os.Stat(absolute)
	if err != nil {
		return Plan{}, err
	}
	if !stat.Mode().IsRegular() || stat.Mode().Perm()&0111 == 0 {
		return Plan{}, fmt.Errorf("not an executable regular file: %s", absolute)
	}
	binary, err := elfinfo.Inspect(absolute)
	if err != nil {
		return Plan{}, err
	}
	if !binary.Glibc() {
		return Plan{}, fmt.Errorf("%w: %s (PT_INTERP=%q)", ErrNotGlibc, absolute, binary.Interpreter)
	}
	selected, err := glruntime.Detect(root)
	if err != nil {
		return Plan{}, err
	}
	loader, err := selected.LoaderFor(binary.Machine)
	if err != nil {
		return Plan{}, err
	}
	loaderELF, err := elfinfo.Inspect(loader)
	if err != nil {
		return Plan{}, fmt.Errorf("runtime loader is not ELF: %w", err)
	}
	if loaderELF.Machine != binary.Machine || loaderELF.Class != elf.ELFCLASS64 || loaderELF.Type != elf.ET_DYN {
		return Plan{}, fmt.Errorf("runtime loader architecture/type does not match %s", binary.Machine)
	}
	libraries := selected.LibraryPaths(binary.Machine)
	if len(libraries) == 0 {
		return Plan{}, fmt.Errorf("no library directories found in runtime %s", selected.Sysroot)
	}
	shim, err := waylandFilter(selected.Root, binary.Machine)
	if err != nil {
		return Plan{}, err
	}
	argv := []string{loader, "--library-path", strings.Join(libraries, ":"), absolute}
	argv = append(argv, args...)
	return Plan{Binary: binary, Runtime: selected, Loader: loader, Args: argv, Preload: shim}, nil
}

// Command runs a plan as the current user, inheriting the environment and cwd.
// It never invokes a shell or obtains additional privileges.
func (p Plan) Command() *exec.Cmd {
	command := exec.Command(p.Loader, p.Args[1:]...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if p.Preload != "" {
		var environment []string
		var prior string
		for _, entry := range os.Environ() {
			if strings.HasPrefix(entry, "LD_PRELOAD=") {
				prior = strings.TrimPrefix(entry, "LD_PRELOAD=")
				continue
			}
			environment = append(environment, entry)
		}
		preload := p.Preload
		if prior != "" {
			preload += ":" + prior
		}
		command.Env = append(environment, "LD_PRELOAD="+preload)
	}
	return command
}

// MachineName is used for human-readable diagnostics.
func MachineName(machine elf.Machine) string {
	if name, err := elfinfo.Architecture(machine); err == nil {
		return name
	}
	return fmt.Sprint(machine)
}

// waylandFilter deliberately reuses the original glenv filter binary rather
// than changing the protocol workaround or copying GPL-3 source into this
// module. The filter is optional in minimal runtime presets.
func waylandFilter(prefix string, machine elf.Machine) (string, error) {
	candidate := filepath.Join(prefix, "lib", "glenv-wayland-filter.so")
	info, err := os.Stat(candidate)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("glenv Wayland filter is not a regular file: %s", candidate)
	}
	actual, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(prefix, actual)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("glenv Wayland filter escapes runtime root")
	}
	binary, err := elfinfo.Inspect(actual)
	if err != nil {
		return "", fmt.Errorf("invalid glenv Wayland filter: %w", err)
	}
	if binary.Type != elf.ET_DYN || binary.Machine != machine || binary.Class != elf.ELFCLASS64 {
		return "", errors.New("glenv Wayland filter ABI does not match target program")
	}
	return actual, nil
}
