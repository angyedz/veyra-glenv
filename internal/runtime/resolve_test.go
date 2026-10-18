package runtime

import (
 "os"
 "path/filepath"
 "testing"
)
func TestDebianAbsoluteSymlink(t *testing.T){
 root:=t.TempDir();dir:=filepath.Join(root,"usr/lib/x86_64-linux-gnu")
 if err:=os.MkdirAll(dir,0755);err!=nil{t.Fatal(err)}
 target:=filepath.Join(dir,"ld-real.so")
 if err:=os.WriteFile(target,[]byte("test"),0755);err!=nil{t.Fatal(err)}
 if err:=os.MkdirAll(filepath.Join(root,"lib64"),0755);err!=nil{t.Fatal(err)}
 if err:=os.Symlink("/usr/lib/x86_64-linux-gnu/ld-real.so",filepath.Join(root,"lib64/ld-linux-x86-64.so.2"));err!=nil{t.Fatal(err)}
 got,err:=resolveInSysroot(root,"lib64/ld-linux-x86-64.so.2")
 if err!=nil||got!=target{t.Fatalf("got %q, err %v",got,err)}
}
func TestRejectTraversal(t *testing.T){
 root:=t.TempDir()
 if err:=os.Symlink("../../outside",filepath.Join(root,"bad"));err!=nil{t.Fatal(err)}
 if _,err:=resolveInSysroot(root,"bad");err==nil{t.Fatal("accepted traversal")}
}
func TestRejectSymlinkLoop(t *testing.T){
 root:=t.TempDir()
 if err:=os.Symlink("b",filepath.Join(root,"a"));err!=nil{t.Fatal(err)}
 if err:=os.Symlink("a",filepath.Join(root,"b"));err!=nil{t.Fatal(err)}
 if _,err:=resolveInSysroot(root,"a");err==nil{t.Fatal("accepted loop")}
}
