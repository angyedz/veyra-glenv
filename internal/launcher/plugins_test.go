package launcher

import (
	"debug/elf"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angyedz/veyra-glenv/internal/elfinfo"
	glruntime "github.com/angyedz/veyra-glenv/internal/runtime"
)

func TestCommandPrefersGlibcPipeWireModules(t *testing.T) {
	root := t.TempDir()
	sysroot := filepath.Join(root, "sysroot")
	spa := filepath.Join(sysroot, "usr/lib/x86_64-linux-gnu/spa-0.2")
	pipewire := filepath.Join(sysroot, "usr/lib/x86_64-linux-gnu/pipewire-0.3")
	for _, d := range []string{spa, pipewire} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("SPA_PLUGIN_DIR", "/host/musl-spa")
	t.Setenv("PIPEWIRE_MODULE_DIR", "/host/musl-pipewire")
	p := Plan{
		Binary:  elfinfo.Binary{Machine: elf.EM_X86_64},
		Runtime: glruntime.Runtime{Root: root, Sysroot: sysroot},
		Loader:  "/bin/true",
		Args:    []string{"/bin/true"},
	}
	command := p.Command()
	observed := map[string]int{}
	for _, env := range command.Env {
		key, value, ok := strings.Cut(env, "=")
		if !ok {
			t.Fatalf("invalid env: %q", env)
		}
		if key == "SPA_PLUGIN_DIR" || key == "PIPEWIRE_MODULE_DIR" {
			observed[key]++
			want := spa
			if key == "PIPEWIRE_MODULE_DIR" {
				want = pipewire
			}
			if value != want {
				t.Errorf("%s=%q, want %q", key, value, want)
			}
		}
	}
	for _, key := range []string{"SPA_PLUGIN_DIR", "PIPEWIRE_MODULE_DIR"} {
		if observed[key] != 1 {
			t.Errorf("expected one %s env, found %d", key, observed[key])
		}
	}
}

func TestCommandUsesRuntimeGraphicsDriversAndDropsHostEGLManifestOverride(t *testing.T) {
	root := t.TempDir()
	sysroot := filepath.Join(root, "sysroot")
	for _, relative := range []string{
		"usr/lib/x86_64-linux-gnu/dri",
		"usr/lib/x86_64-linux-gnu/egl",
		"usr/share/glvnd/egl_vendor.d",
		"usr/share/vulkan/icd.d",
	} {
		if err := os.MkdirAll(filepath.Join(sysroot, relative), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for key, value := range map[string]string{
		"LIBGL_DRIVERS_PATH":             "/host/musl-dri",
		"EGL_DRIVERS_PATH":               "/host/musl-egl",
		"__EGL_VENDOR_LIBRARY_DIRS":      "/host/musl-egl-vendors",
		"__EGL_VENDOR_LIBRARY_FILENAMES": "/host/musl-egl.json",
		"VK_DRIVER_FILES":                "/host/musl-vulkan",
		"VK_ICD_FILENAMES":               "/host/musl-vulkan-old",
	} {
		t.Setenv(key, value)
	}
	p := Plan{
		Binary:  elfinfo.Binary{Machine: elf.EM_X86_64},
		Runtime: glruntime.Runtime{Root: root, Sysroot: sysroot},
		Loader:  "/bin/true",
		Args:    []string{"/bin/true"},
	}
	command := p.Command()
	observed := map[string]string{}
	for _, entry := range command.Env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			t.Fatalf("invalid env: %q", entry)
		}
		observed[key] = value
	}
	for _, key := range []string{"LIBGL_DRIVERS_PATH", "EGL_DRIVERS_PATH", "__EGL_VENDOR_LIBRARY_DIRS", "VK_DRIVER_FILES", "VK_ICD_FILENAMES"} {
		if value := observed[key]; value == "" || strings.Contains(value, "/host/") || !strings.Contains(value, sysroot) {
			t.Errorf("%s did not resolve inside the runtime: %q", key, value)
		}
	}
	if _, found := observed["__EGL_VENDOR_LIBRARY_FILENAMES"]; found {
		t.Fatal("inherited host EGL manifest list was not removed")
	}
}
