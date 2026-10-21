package runtime

import (
	"debug/elf"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphicsEnvironmentUsesOnlySysrootPaths(t *testing.T) {
	root := t.TempDir()
	sysroot := filepath.Join(root, "sysroot")
	for _, relative := range []string{
		"usr/lib/x86_64-linux-gnu/dri",
		"usr/lib/x86_64-linux-gnu/egl",
		"etc/glvnd/egl_vendor.d",
		"usr/share/glvnd/egl_vendor.d",
		"usr/share/vulkan/icd.d",
		"usr/lib/x86_64-linux-gnu/vulkan/icd.d",
	} {
		if err := os.MkdirAll(filepath.Join(sysroot, relative), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, relative := range []string{
		"usr/share/vulkan/icd.d/nouveau_icd.json",
		"usr/lib/x86_64-linux-gnu/vulkan/icd.d/lvp_icd.json",
	} {
		if err := os.WriteFile(filepath.Join(sysroot, relative), []byte(`{"ICD":{}}`), 0644); err != nil {
			t.Fatal(err)
		}
	}
	environment := (Runtime{Root: root, Sysroot: sysroot}).GraphicsEnvironment(elf.EM_X86_64)
	for _, key := range []string{"LIBGL_DRIVERS_PATH", "EGL_DRIVERS_PATH", "__EGL_VENDOR_LIBRARY_DIRS"} {
		value := environment[key]
		if value == "" {
			t.Fatalf("missing %s", key)
		}
		for _, directory := range strings.Split(value, string(os.PathListSeparator)) {
			if !within(sysroot, directory) {
				t.Fatalf("%s contains a path outside the runtime: %q", key, directory)
			}
		}
	}
	for _, key := range []string{"VK_DRIVER_FILES", "VK_ICD_FILENAMES"} {
		value := environment[key]
		if value == "" {
			t.Fatalf("missing %s", key)
		}
		for _, manifest := range strings.Split(value, string(os.PathListSeparator)) {
			if !within(sysroot, manifest) || filepath.Ext(manifest) != ".json" {
				t.Fatalf("%s contains a non-runtime manifest path: %q", key, manifest)
			}
			if info, err := os.Stat(manifest); err != nil || !info.Mode().IsRegular() {
				t.Fatalf("%s references a missing manifest: %q %v", key, manifest, err)
			}
		}
	}
	if environment["VK_DRIVER_FILES"] != environment["VK_ICD_FILENAMES"] {
		t.Fatal("new and legacy Vulkan variables should point at the same runtime manifests")
	}
	if !strings.Contains(environment["__EGL_VENDOR_LIBRARY_DIRS"], filepath.Join(sysroot, "etc", "glvnd", "egl_vendor.d")) ||
		!strings.Contains(environment["__EGL_VENDOR_LIBRARY_DIRS"], filepath.Join(sysroot, "usr", "share", "glvnd", "egl_vendor.d")) {
		t.Fatalf("did not include both runtime EGL manifest directories: %+v", environment)
	}
}

func TestGraphicsEnvironmentDisablesHostFallbackWhenRuntimeHasNoDrivers(t *testing.T) {
	sysroot := filepath.Join(t.TempDir(), "sysroot")
	if err := os.MkdirAll(sysroot, 0755); err != nil {
		t.Fatal(err)
	}
	environment := (Runtime{Sysroot: sysroot}).GraphicsEnvironment(elf.EM_X86_64)
	for _, key := range []string{"LIBGL_DRIVERS_PATH", "EGL_DRIVERS_PATH", "__EGL_VENDOR_LIBRARY_DIRS"} {
		if !within(sysroot, environment[key]) {
			t.Fatalf("%s did not fail closed inside the runtime: %q", key, environment[key])
		}
	}
	for _, key := range []string{"VK_DRIVER_FILES", "VK_ICD_FILENAMES"} {
		if !within(sysroot, environment[key]) || filepath.Ext(environment[key]) != ".json" {
			t.Fatalf("%s did not fail closed to an absent runtime manifest: %q", key, environment[key])
		}
		if _, err := os.Stat(environment[key]); !os.IsNotExist(err) {
			t.Fatalf("%s fallback unexpectedly exists: %q %v", key, environment[key], err)
		}
	}
}

func TestPluginEnvironmentUsesOnlySysrootDirectories(t *testing.T) {
	root := t.TempDir()
	sysroot := filepath.Join(root, "sysroot")
	for _, relative := range []string{
		"usr/lib/x86_64-linux-gnu/spa-0.2",
		"usr/lib/x86_64-linux-gnu/pipewire-0.3",
		"usr/lib/x86_64-linux-gnu/girepository-1.0",
		"usr/share/pipewire",
	} {
		if err := os.MkdirAll(filepath.Join(sysroot, relative), 0755); err != nil {
			t.Fatal(err)
		}
	}
	selected := Runtime{Root: root, Sysroot: sysroot}
	vars := selected.PluginEnvironment(elf.EM_X86_64)
	for _, key := range []string{"SPA_PLUGIN_DIR", "PIPEWIRE_MODULE_DIR", "PIPEWIRE_CONFIG_DIR", "GI_TYPELIB_PATH"} {
		if vars[key] == "" {
			t.Fatalf("missing %s: %+v", key, vars)
		}
		if !within(sysroot, vars[key]) {
			t.Fatalf("directory escapes sysroot: %+v", vars)
		}
	}
	if selected.PluginEnvironment(elf.EM_AARCH64)["SPA_PLUGIN_DIR"] != "" {
		t.Fatal("used x86_64 plugins for arm64")
	}
	if len(selected.PluginEnvironment(elf.EM_NONE)) != 0 {
		t.Fatal("unexpected plugin paths for unknown arch")
	}
}

func TestPluginEnvironmentRejectsEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	sysroot := filepath.Join(root, "sysroot")
	dir := filepath.Join(sysroot, "usr/lib/x86_64-linux-gnu")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "spa-0.2")); err != nil {
		t.Fatal(err)
	}
	selected := Runtime{Sysroot: sysroot}
	if actual := selected.PluginEnvironment(elf.EM_X86_64)["SPA_PLUGIN_DIR"]; actual != "" {
		t.Fatalf("accepted escaping SPA module directory: %s", actual)
	}
}

func TestGdkPixbufEnvironmentPrefersDebian(t *testing.T) {
	root := t.TempDir()
	sysroot := filepath.Join(root, "sysroot")
	dir := filepath.Join(sysroot, "usr/lib/x86_64-linux-gnu/gdk-pixbuf-2.0/2.10.0")
	if err := os.MkdirAll(filepath.Join(dir, "loaders"), 0755); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(dir, "loaders.cache")
	if err := os.WriteFile(cache, []byte("Debian fixture"), 0644); err != nil {
		t.Fatal(err)
	}
	selected := Runtime{Root: root, Sysroot: sysroot}
	env := selected.PluginEnvironment(elf.EM_X86_64)
	if env["GDK_PIXBUF_MODULEDIR"] != filepath.Join(dir, "loaders") {
		t.Fatalf("wrong pixbuf module dir: %+v", env)
	}
	if env["GDK_PIXBUF_MODULE_FILE"] != cache {
		t.Fatalf("wrong pixbuf cache: %+v", env)
	}
	if err := os.Remove(cache); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "loaders.cache")
	if err := os.WriteFile(outside, []byte("foreign"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, cache); err != nil {
		t.Fatal(err)
	}
	if actual := selected.PluginEnvironment(elf.EM_X86_64)["GDK_PIXBUF_MODULE_FILE"]; actual != "" {
		t.Fatalf("accepted escaping pixbuf cache symlink: %s", actual)
	}
}

func TestGtkResourcesPreferDebianWithoutLosingHostData(t *testing.T) {
	root := t.TempDir()
	sysroot := filepath.Join(root, "sysroot")
	share := filepath.Join(sysroot, "usr/share")
	if err := os.MkdirAll(share, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_DATA_DIRS", "/host/share:/usr/share")
	selected := Runtime{Root: root, Sysroot: sysroot}
	value := selected.PluginEnvironment(elf.EM_X86_64)["XDG_DATA_DIRS"]
	if value != share+":/host/share:/usr/share" {
		t.Fatalf("wrong XDG_DATA_DIRS: %q", value)
	}
}
