package deployment

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	glruntime "github.com/angyedz/veyra-glenv/internal/runtime"
)

var graphicsManifestDirectories = []string{
	"usr/share/vulkan/icd.d",
	"etc/vulkan/icd.d",
	"usr/lib/x86_64-linux-gnu/vulkan/icd.d",
	"usr/lib/aarch64-linux-gnu/vulkan/icd.d",
	"usr/share/glvnd/egl_vendor.d",
	"etc/glvnd/egl_vendor.d",
}

var graphicsLibraryDirectories = []string{
	"usr/lib/x86_64-linux-gnu",
	"lib/x86_64-linux-gnu",
	"usr/lib/aarch64-linux-gnu",
	"lib/aarch64-linux-gnu",
	"usr/lib64",
	"lib64",
	"usr/lib",
	"lib",
	"usr/local/lib",
}

// normalizeGraphicsManifests rewrites Mesa/Vulkan manifests in a staged copy
// so they cannot resolve absolute /usr paths against the musl host. Vulkan
// manifests get a sysroot-relative library path. GLVND uses a bare filename,
// which the glenv loader resolves from its explicit runtime library paths.
func normalizeGraphicsManifests(sysroot string) error {
	runtime := glruntime.Runtime{Sysroot: sysroot}
	for _, relativeDirectory := range graphicsManifestDirectories {
		directory := runtime.DirectoryInSysroot(relativeDirectory)
		if directory == "" {
			continue
		}
		entries, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			filename := filepath.Join(directory, entry.Name())
			info, err := os.Lstat(filename)
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("graphics manifest is not a regular file: %s", filename)
			}
			data, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			var document map[string]json.RawMessage
			if err := json.Unmarshal(data, &document); err != nil {
				return fmt.Errorf("invalid graphics manifest %s: %w", filename, err)
			}
			var icd map[string]json.RawMessage
			if raw := document["ICD"]; len(raw) == 0 {
				continue
			} else if err := json.Unmarshal(raw, &icd); err != nil {
				return fmt.Errorf("invalid ICD section in %s: %w", filename, err)
			}
			var libraryPath string
			if raw := icd["library_path"]; len(raw) == 0 {
				return fmt.Errorf("graphics manifest lacks ICD.library_path: %s", filename)
			} else if err := json.Unmarshal(raw, &libraryPath); err != nil || libraryPath == "" {
				return fmt.Errorf("invalid ICD.library_path in %s", filename)
			}
			manifestRelative, err := filepath.Rel(sysroot, filename)
			if err != nil {
				return err
			}
			resolvedLibrary, err := resolveGraphicsLibrary(runtime, filepath.ToSlash(manifestRelative), libraryPath)
			if err != nil {
				return fmt.Errorf("unsafe or missing graphics library in %s: %w", filename, err)
			}
			libraryRelative, err := filepath.Rel(sysroot, resolvedLibrary)
			if err != nil {
				return err
			}
			libraryRelative = filepath.ToSlash(libraryRelative)
			normalized := libraryRelative
			if strings.Contains(relativeDirectory, "glvnd/egl_vendor.d") {
				if !containsString(graphicsLibraryDirectories, path.Dir(libraryRelative)) {
					return fmt.Errorf("GLVND library is outside glenv loader search paths: %s", libraryRelative)
				}
				normalized = path.Base(libraryRelative)
			} else {
				var relative string
				relative, err = filepath.Rel(filepath.FromSlash(path.Dir(filepath.ToSlash(manifestRelative))), filepath.FromSlash(libraryRelative))
				if err != nil {
					return err
				}
				normalized = filepath.ToSlash(relative)
			}
			if normalized == libraryPath {
				continue
			}
			encodedPath, err := json.Marshal(normalized)
			if err != nil {
				return err
			}
			icd["library_path"] = encodedPath
			encodedICD, err := json.Marshal(icd)
			if err != nil {
				return err
			}
			document["ICD"] = encodedICD
			updated, err := json.MarshalIndent(document, "", "  ")
			if err != nil {
				return err
			}
			updated = append(updated, '\n')
			if err := replaceFile(filename, updated, info.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func resolveGraphicsLibrary(runtime glruntime.Runtime, manifestRelative, libraryPath string) (string, error) {
	if strings.ContainsRune(libraryPath, '\\') {
		return "", fmt.Errorf("unsupported path separator in %q", libraryPath)
	}
	if path.IsAbs(libraryPath) {
		libraryPath = strings.TrimPrefix(path.Clean(libraryPath), "/")
	} else if strings.Contains(libraryPath, "/") {
		libraryPath = path.Clean(path.Join(path.Dir(manifestRelative), libraryPath))
	} else {
		for _, directory := range graphicsLibraryDirectories {
			if resolved := runtime.FileInSysroot(filepath.FromSlash(path.Join(directory, libraryPath))); resolved != "" {
				return resolved, nil
			}
		}
		return "", fmt.Errorf("library %q is absent from glenv loader paths", libraryPath)
	}
	if libraryPath == "." || libraryPath == ".." || strings.HasPrefix(libraryPath, "../") {
		return "", fmt.Errorf("library path escapes glenv: %q", libraryPath)
	}
	if resolved := runtime.FileInSysroot(filepath.FromSlash(libraryPath)); resolved != "" {
		return resolved, nil
	}
	return "", fmt.Errorf("library %q is absent from glenv", libraryPath)
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func replaceFile(filename string, contents []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(filename), ".veyra-graphics-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, filename)
}
