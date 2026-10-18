package elfinfo

import (
 "debug/elf"
 "os"
 "path/filepath"
 "testing"
)

func TestClassification(t *testing.T){
 cases:=[]struct{b Binary; want bool}{
  {Binary{Machine:elf.EM_X86_64,Class:elf.ELFCLASS64,Interpreter:"/lib64/ld-linux-x86-64.so.2"},true},
  {Binary{Machine:elf.EM_AARCH64,Class:elf.ELFCLASS64,Interpreter:"/lib/ld-linux-aarch64.so.1"},true},
  {Binary{Machine:elf.EM_X86_64,Class:elf.ELFCLASS64,Interpreter:"/lib/ld-musl-x86_64.so.1"},false},
  {Binary{Machine:elf.EM_X86_64,Class:elf.ELFCLASS64},false},
  {Binary{Machine:elf.EM_386,Class:elf.ELFCLASS32,Interpreter:"/lib/ld-linux.so.2"},false},
 }
 for _,tc:=range cases{if got:=tc.b.Glibc();got!=tc.want{t.Errorf("%+v: got %t want %t",tc.b,got,tc.want)}}
}
func TestRejectNonELF(t *testing.T){
 path:=filepath.Join(t.TempDir(),"script")
 if err:=os.WriteFile(path,[]byte("#!/bin/sh\n"),0755);err!=nil{t.Fatal(err)}
 if _,err:=Inspect(path);err==nil{t.Fatal("expected ELF error")}
}
