package runtime

import (
	"debug/elf"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GraphicsEnvironment confines Mesa, EGL and Vulkan driver discovery to the
// selected glenv sysroot. When a runtime does not contain a driver directory,
// the corresponding path points at a reserved, absent location in the
// read-only runtime so the loader cannot fall back to musl-host drivers.
func (r Runtime) GraphicsEnvironment(machine elf.Machine) map[string]string {
	var abi string
	switch machine {
	case elf.EM_X86_64:
		abi = "x86_64-linux-gnu"
	case elf.EM_AARCH64:
		abi = "aarch64-linux-gnu"
	default:
		return nil
	}

	vulkanManifests := r.vulkanManifestFiles(abi)
	return map[string]string{
		"LIBGL_DRIVERS_PATH": r.directoryList("mesa-dri", "usr/lib/"+abi+"/dri", "usr/lib/dri"),
		"EGL_DRIVERS_PATH":   r.directoryList("mesa-egl", "usr/lib/"+abi+"/egl", "usr/lib/egl"),
		"__EGL_VENDOR_LIBRARY_DIRS": r.directoryList("egl-vendors",
			"etc/glvnd/egl_vendor.d", "usr/share/glvnd/egl_vendor.d"),
		// Both Vulkan variables accept manifest filenames. Use the exact files
		// for compatibility with older loaders that do not accept directories.
		"VK_DRIVER_FILES":  vulkanManifests,
		"VK_ICD_FILENAMES": vulkanManifests,
	}
}

func (r Runtime) vulkanManifestFiles(abi string) string {
	root, err := filepath.Abs(r.Sysroot)
	if err != nil {
		return filepath.Join(r.Sysroot, ".veyra-disabled", "vulkan-icds", "missing.json")
	}
	seen := make(map[string]bool)
	var manifests []string
	for _, candidate := range []string{
		"usr/share/vulkan/icd.d",
		"etc/vulkan/icd.d",
		"usr/lib/" + abi + "/vulkan/icd.d",
	} {
		directory := r.DirectoryInSysroot(candidate)
		if directory == "" {
			continue
		}
		entries, err := os.ReadDir(directory)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			relative, err := filepath.Rel(root, filepath.Join(directory, entry.Name()))
			if err != nil {
				continue
			}
			manifest := r.FileInSysroot(relative)
			if manifest == "" || seen[manifest] {
				continue
			}
			seen[manifest] = true
			manifests = append(manifests, manifest)
		}
	}
	if len(manifests) == 0 {
		return filepath.Join(root, ".veyra-disabled", "vulkan-icds", "missing.json")
	}
	sort.Strings(manifests)
	return strings.Join(manifests, string(os.PathListSeparator))
}

func (r Runtime) directoryList(disabledName string, candidates ...string) string {
	seen := make(map[string]bool)
	var directories []string
	for _, candidate := range candidates {
		resolved := r.DirectoryInSysroot(candidate)
		if resolved == "" || seen[resolved] {
			continue
		}
		seen[resolved] = true
		directories = append(directories, resolved)
	}
	if len(directories) == 0 {
		return filepath.Join(r.Sysroot, ".veyra-disabled", disabledName)
	}
	return strings.Join(directories, string(os.PathListSeparator))
}
