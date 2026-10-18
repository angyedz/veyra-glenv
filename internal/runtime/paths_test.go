package runtime

import (
 "debug/elf"
 "os"
 "path/filepath"
 "testing"
)

func TestDebianSysroot(t *testing.T){
 prefix:=t.TempDir()
 lib:=filepath.Join(prefix,"sysroot","usr","lib","x86_64-linux-gnu")
 if err:=os.MkdirAll(lib,0755);err!=nil{t.Fatal(err)}
 loader:=filepath.Join(lib,"ld-linux-x86-64.so.2")
 if err:=os.WriteFile(loader,[]byte("fake"),0755);err!=nil{t.Fatal(err)}
 r,err:=Detect(prefix);if err!=nil{t.Fatal(err)}
 if r.Sysroot!=filepath.Join(prefix,"sysroot"){t.Fatalf("sysroot %q",r.Sysroot)}
 got,err:=r.LoaderFor(elf.EM_X86_64);if err!=nil||got!=loader{t.Fatalf("loader %q: %v",got,err)}
 if len(r.LibraryPaths(elf.EM_X86_64))==0{t.Fatal("missing library paths")}
 if _,err:=r.LoaderFor(elf.EM_AARCH64);err==nil{t.Fatal("cross-architecture loader unexpectedly found")}
}
func TestRejectEscapingLoaderSymlink(t *testing.T){
 root:=t.TempDir()
 outside:=filepath.Join(t.TempDir(),"ld-linux-x86-64.so.2")
 if err:=os.WriteFile(outside,[]byte("fake"),0755);err!=nil{t.Fatal(err)}
 if err:=os.MkdirAll(filepath.Join(root,"lib64"),0755);err!=nil{t.Fatal(err)}
 if err:=os.Symlink(outside,filepath.Join(root,"lib64","ld-linux-x86-64.so.2"));err!=nil{t.Fatal(err)}
 r:=Runtime{Root:root,Sysroot:root}
 if _,err:=r.LoaderFor(elf.EM_X86_64);err==nil{t.Fatal("accepted escaping loader symlink")}
}
