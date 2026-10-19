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
    resolved,err:=resolveInSysroot(r.Sysroot,relative)
    if err!=nil||!within(r.Sysroot,resolved){return ""}
    info,err:=os.Stat(resolved)
    if err!=nil||!info.IsDir(){return ""}
    return resolved
}

// PluginEnvironment selects glibc-native PipeWire, SPA and GTK/GI metadata.
// Only existing directories inside the selected Debian runtime are published.
// They are runtime-owned paths, never Alpine/musl host module directories.
func (r Runtime) PluginEnvironment(machine elf.Machine) map[string]string {
    var abi string
    switch machine {
    case elf.EM_X86_64: abi="x86_64-linux-gnu"
    case elf.EM_AARCH64: abi="aarch64-linux-gnu"
    default: return nil
    }
    options:=map[string][]string{
        "SPA_PLUGIN_DIR": {"usr/lib/"+abi+"/spa-0.2", "usr/lib/spa-0.2"},
        "PIPEWIRE_MODULE_DIR": {"usr/lib/"+abi+"/pipewire-0.3","usr/lib/pipewire-0.3"},
        "PIPEWIRE_CONFIG_DIR": {"usr/share/pipewire","etc/pipewire"},
        "SPA_DATA_DIR": {"usr/share/spa-0.2"},
        "GI_TYPELIB_PATH": {"usr/lib/"+abi+"/girepository-1.0"},
    }
    result:=make(map[string]string)
    for name,candidates:=range options{
        for _,relative:=range candidates {
            if found:=r.DirectoryInSysroot(relative);found!=""{
                result[name]=found
                break
            }
        }
    }
    return result
}
