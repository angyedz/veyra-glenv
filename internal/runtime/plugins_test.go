package runtime

import (
 "debug/elf"
 "os"
 "path/filepath"
 "testing"
)

func TestPluginEnvironmentUsesOnlySysrootDirectories(t *testing.T) {
 root:=t.TempDir()
 sysroot:=filepath.Join(root,"sysroot")
 for _,relative:=range []string{
  "usr/lib/x86_64-linux-gnu/spa-0.2",
  "usr/lib/x86_64-linux-gnu/pipewire-0.3",
  "usr/lib/x86_64-linux-gnu/girepository-1.0",
  "usr/share/pipewire",
 }{
  if err:=os.MkdirAll(filepath.Join(sysroot,relative),0755);err!=nil{t.Fatal(err)}
 }
 selected:=Runtime{Root:root,Sysroot:sysroot}
 vars:=selected.PluginEnvironment(elf.EM_X86_64)
 for _,key:=range []string{"SPA_PLUGIN_DIR","PIPEWIRE_MODULE_DIR","PIPEWIRE_CONFIG_DIR","GI_TYPELIB_PATH"}{
  if vars[key]==""{t.Fatalf("missing %s: %+v",key,vars)}
  if !within(sysroot,vars[key]){t.Fatalf("directory escapes sysroot: %+v",vars)}
 }
 if selected.PluginEnvironment(elf.EM_AARCH64)["SPA_PLUGIN_DIR"]!=""{t.Fatal("used x86_64 plugins for arm64")}
 if len(selected.PluginEnvironment(elf.EM_NONE))!=0{t.Fatal("unexpected plugin paths for unknown arch")}
}

func TestPluginEnvironmentRejectsEscapingSymlink(t *testing.T){
 root:=t.TempDir()
 sysroot:=filepath.Join(root,"sysroot")
 dir:=filepath.Join(sysroot,"usr/lib/x86_64-linux-gnu")
 if err:=os.MkdirAll(dir,0755);err!=nil{t.Fatal(err)}
 outside:=t.TempDir()
 if err:=os.Symlink(outside,filepath.Join(dir,"spa-0.2"));err!=nil{t.Fatal(err)}
 selected:=Runtime{Sysroot:sysroot}
 if actual:=selected.PluginEnvironment(elf.EM_X86_64)["SPA_PLUGIN_DIR"];actual!=""{
  t.Fatalf("accepted escaping SPA module directory: %s",actual)
 }
}

func TestGdkPixbufEnvironmentPrefersDebian(t *testing.T){
 root:=t.TempDir()
 sysroot:=filepath.Join(root,"sysroot")
 dir:=filepath.Join(sysroot,"usr/lib/x86_64-linux-gnu/gdk-pixbuf-2.0/2.10.0")
 if err:=os.MkdirAll(filepath.Join(dir,"loaders"),0755);err!=nil{t.Fatal(err)}
 cache:=filepath.Join(dir,"loaders.cache")
 if err:=os.WriteFile(cache,[]byte("Debian fixture"),0644);err!=nil{t.Fatal(err)}
 selected:=Runtime{Root:root,Sysroot:sysroot}
 env:=selected.PluginEnvironment(elf.EM_X86_64)
 if env["GDK_PIXBUF_MODULEDIR"]!=filepath.Join(dir,"loaders"){t.Fatalf("wrong pixbuf module dir: %+v",env)}
 if env["GDK_PIXBUF_MODULE_FILE"]!=cache{t.Fatalf("wrong pixbuf cache: %+v",env)}
 if err:=os.Remove(cache);err!=nil{t.Fatal(err)}
 outside:=filepath.Join(t.TempDir(),"loaders.cache")
 if err:=os.WriteFile(outside,[]byte("foreign"),0644);err!=nil{t.Fatal(err)}
 if err:=os.Symlink(outside,cache);err!=nil{t.Fatal(err)}
 if actual:=selected.PluginEnvironment(elf.EM_X86_64)["GDK_PIXBUF_MODULE_FILE"];actual!=""{
  t.Fatalf("accepted escaping pixbuf cache symlink: %s",actual)
 }
}

func TestGtkResourcesPreferDebianWithoutLosingHostData(t *testing.T){
 root:=t.TempDir()
 sysroot:=filepath.Join(root,"sysroot")
 share:=filepath.Join(sysroot,"usr/share")
 if err:=os.MkdirAll(share,0755);err!=nil{t.Fatal(err)}
 t.Setenv("XDG_DATA_DIRS","/host/share:/usr/share")
 selected:=Runtime{Root:root,Sysroot:sysroot}
 value:=selected.PluginEnvironment(elf.EM_X86_64)["XDG_DATA_DIRS"]
 if value!=share+":/host/share:/usr/share"{t.Fatalf("wrong XDG_DATA_DIRS: %q",value)}
}
