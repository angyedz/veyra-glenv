package runtime

import (
 "debug/elf"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"
)

var ErrNotFound = errors.New("glenv runtime not found")

// Runtime identifies a glibc sysroot without changing the musl host.
type Runtime struct {
 Root string
 Sysroot string
 Loader string // compatibility: first available supported loader
}

func Detect(root string) (Runtime,error) {
 if root=="" { return Runtime{},ErrNotFound }
 absolute,err:=filepath.Abs(root); if err!=nil{return Runtime{},err}
 absolute,err=filepath.EvalSymlinks(absolute);if err!=nil{return Runtime{},fmt.Errorf("%w: %v",ErrNotFound,err)}
 info,err:=os.Stat(absolute);if err!=nil{return Runtime{},err}
 if !info.IsDir(){return Runtime{},fmt.Errorf("not a runtime directory: %s",absolute)}
 sysroot:=absolute
 candidate:=filepath.Join(absolute,"sysroot")
 if stat,err:=os.Stat(candidate);err==nil&&stat.IsDir(){
  sysroot,err=filepath.EvalSymlinks(candidate);if err!=nil{return Runtime{},err}
 }
 r:=Runtime{Root:absolute,Sysroot:sysroot}
 for _,arch:=range []elf.Machine{elf.EM_X86_64,elf.EM_AARCH64}{
  if loader,err:=r.LoaderFor(arch);err==nil {r.Loader=loader;return r,nil}
 }
 return Runtime{},fmt.Errorf("%w: no supported loader in %s",ErrNotFound,sysroot)
}

func (r Runtime) LoaderFor(machine elf.Machine)(string,error){
 var candidates []string
 switch machine {
 case elf.EM_X86_64:
  candidates=[]string{"usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2","lib/x86_64-linux-gnu/ld-linux-x86-64.so.2","lib64/ld-linux-x86-64.so.2","lib/ld-linux-x86-64.so.2"}
 case elf.EM_AARCH64:
  candidates=[]string{"usr/lib/aarch64-linux-gnu/ld-linux-aarch64.so.1","lib/aarch64-linux-gnu/ld-linux-aarch64.so.1","lib/ld-linux-aarch64.so.1"}
 default:return "",fmt.Errorf("unsupported machine: %s",machine)
 }
 for _,rel:=range candidates{
  resolved,err:=filepath.EvalSymlinks(filepath.Join(r.Sysroot,rel))
  if err!=nil||!within(r.Sysroot,resolved){continue}
  stat,err:=os.Stat(resolved)
  if err==nil&&stat.Mode().IsRegular(){return resolved,nil}
 }
 return "",fmt.Errorf("%w: loader for %s in %s",ErrNotFound,machine,r.Sysroot)
}

func (r Runtime) LibraryPaths(machine elf.Machine)[]string{
 var abi []string
 switch machine{
 case elf.EM_X86_64:abi=[]string{"usr/lib/x86_64-linux-gnu","lib/x86_64-linux-gnu"}
 case elf.EM_AARCH64:abi=[]string{"usr/lib/aarch64-linux-gnu","lib/aarch64-linux-gnu"}
 default:return nil
 }
 paths:=[]string{};seen:=map[string]bool{}
 for _,rel:=range append(abi,"usr/lib64","lib64","usr/lib","lib","usr/local/lib"){
  resolved,err:=filepath.EvalSymlinks(filepath.Join(r.Sysroot,rel))
  if err!=nil||!within(r.Sysroot,resolved)||seen[resolved]{continue}
  stat,err:=os.Stat(resolved)
  if err==nil&&stat.IsDir(){seen[resolved]=true;paths=append(paths,resolved)}
 }
 return paths
}

func within(root,path string)bool{
 rel,err:=filepath.Rel(root,path)
 return err==nil&&rel!=".."&&!strings.HasPrefix(rel,".."+string(filepath.Separator))
}
