package launcher

import (
 "debug/elf"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestOptionalWaylandFilter(t *testing.T) {
 root:=t.TempDir()
 shim,err:=waylandFilter(root,elf.EM_X86_64)
 if err!=nil||shim!=""{t.Fatalf("optional filter: %q, %v",shim,err)}
}

func TestRejectEscapingWaylandFilter(t *testing.T) {
 root:=t.TempDir()
 if err:=os.Mkdir(filepath.Join(root,"lib"),0755);err!=nil{t.Fatal(err)}
 outside:=filepath.Join(t.TempDir(),"filter.so")
 if err:=os.WriteFile(outside,[]byte("invalid"),0644);err!=nil{t.Fatal(err)}
 if err:=os.Symlink(outside,filepath.Join(root,"lib","glenv-wayland-filter.so"));err!=nil{t.Fatal(err)}
 if _,err:=waylandFilter(root,elf.EM_X86_64);err==nil{t.Fatal("accepted filter symlink escaping runtime")}
}
func TestRejectInvalidWaylandFilter(t *testing.T) {
 root:=t.TempDir()
 if err:=os.Mkdir(filepath.Join(root,"lib"),0755);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(root,"lib","glenv-wayland-filter.so"),[]byte("not ELF"),0644);err!=nil{t.Fatal(err)}
 if _,err:=waylandFilter(root,elf.EM_X86_64);err==nil{t.Fatal("accepted non-ELF filter")}
}

func TestPreloadIncludesPriorEnvironment(t *testing.T) {
 t.Setenv("LD_PRELOAD","/some/app-helper.so")
 p:=Plan{Loader:"/bin/true",Args:[]string{"/bin/true"},Preload:"/opt/glenv/lib/glenv-wayland-filter.so"}
 command:=p.Command()
 matches:=0
 for _,e:=range command.Env{
  if strings.HasPrefix(e,"LD_PRELOAD="){
   matches++
   if e!="LD_PRELOAD=/opt/glenv/lib/glenv-wayland-filter.so:/some/app-helper.so"{t.Fatalf("preload environment: %q",e)}
  }
 }
 if matches!=1{t.Fatalf("expected one LD_PRELOAD variable, got %d",matches)}
}
