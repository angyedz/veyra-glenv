package runtime

import (
	"debug/elf"
	"os"
)

// DirectoryInSysroot resolves Debian absolute/relative symlinks as paths
// INSIDE this sysroot. It returns an empty string if the directory is absent
// or if resolution would escape the selected runtime.
func (r Runtime) DirectoryInSysroot(relative string) string {
	if r.Sysroot == "" {
		return ""
	}
	resolved, err := resolveInSysroot(r.Sysroot, relative)
	if err != nil || !within(r.Sysroot, resolved) {
		return ""
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return ""
	}
	return resolved
}

// FileInSysroot resolves a regular metadata file within the selected Debian
// sysroot, without following absolute symlinks into the musl host.
func (r Runtime) FileInSysroot(relative string) string {
	if r.Sysroot == "" {
		return ""
	}
	resolved, err := resolveInSysroot(r.Sysroot, relative)
	if err != nil || !within(r.Sysroot, resolved) {
		return ""
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	return resolved
}

// PluginEnvironment selects glibc-native graphics, PipeWire, SPA and GTK/GI
// paths. Plugin directories come only from the selected Debian runtime; GPU
// paths use a disabled runtime-local location when the runtime lacks drivers,
// preventing accidental discovery of Alpine/musl host modules.
func (r Runtime) PluginEnvironment(machine elf.Machine) map[string]string {
	var abi string
	switch machine {
	case elf.EM_X86_64:
		abi = "x86_64-linux-gnu"
	case elf.EM_AARCH64:
		abi = "aarch64-linux-gnu"
	default:
		return nil
	}
	options := map[string][]string{
		"SPA_PLUGIN_DIR":       {"usr/lib/" + abi + "/spa-0.2", "usr/lib/spa-0.2"},
		"PIPEWIRE_MODULE_DIR":  {"usr/lib/" + abi + "/pipewire-0.3", "usr/lib/pipewire-0.3"},
		"PIPEWIRE_CONFIG_DIR":  {"usr/share/pipewire", "etc/pipewire"},
		"SPA_DATA_DIR":         {"usr/share/spa-0.2"},
		"GI_TYPELIB_PATH":      {"usr/lib/" + abi + "/girepository-1.0"},
		"GIO_EXTRA_MODULES":    {"usr/lib/" + abi + "/gio/modules"},
		"GSETTINGS_SCHEMA_DIR": {"usr/share/glib-2.0/schemas"},
		"GDK_PIXBUF_MODULEDIR": {"usr/lib/" + abi + "/gdk-pixbuf-2.0/2.10.0/loaders"},
	}
	result := make(map[string]string)
	for name, value := range r.GraphicsEnvironment(machine) {
		result[name] = value
	}
	for name, candidates := range options {
		for _, relative := range candidates {
			if found := r.DirectoryInSysroot(relative); found != "" {
				result[name] = found
				break
			}
		}
	}
	// A loader cache is needed for GTK's built-in icons and application
	// images. Its contents must be relocatable or generated for the exact
	// runtime mountpoint: Debian's stock cache may embed /usr/lib paths.
	if file := r.FileInSysroot("usr/lib/" + abi + "/gdk-pixbuf-2.0/2.10.0/loaders.cache"); file != "" {
		result["GDK_PIXBUF_MODULE_FILE"] = file
	}
	// GTK icons, MIME metadata and other non-executable desktop resources
	// also live under Debian's /usr/share. Without this, GTK sees only
	// Alpine's icon themes and falls back to resources that can fail to load.
	if share := r.DirectoryInSysroot("usr/share"); share != "" {
		inherited := os.Getenv("XDG_DATA_DIRS")
		if inherited == "" {
			inherited = "/usr/local/share:/usr/share"
		}
		result["XDG_DATA_DIRS"] = share + ":" + inherited
	}
	if xdg := r.DirectoryInSysroot("etc/xdg"); xdg != "" {
		inherited := os.Getenv("XDG_CONFIG_DIRS")
		if inherited == "" {
			inherited = "/etc/xdg"
		}
		result["XDG_CONFIG_DIRS"] = xdg + ":" + inherited
	}
	return result
}
