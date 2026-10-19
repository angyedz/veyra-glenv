package deployment

import (
 "encoding/json"
 "os"
 "path/filepath"
 "testing"
)

func fixture(t *testing.T,parent,version string)string{
 t.Helper()
 dir:=filepath.Join(parent,"staging-"+version)
 if err:=os.MkdirAll(filepath.Join(dir,"sysroot"),0755);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"sysroot","libc.so"),[]byte(version),0644);err!=nil{t.Fatal(err)}
 m,err:=Inventory(dir,version);if err!=nil{t.Fatal(err)}
 data,err:=json.Marshal(m);if err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"glenv-manifest.json"),data,0644);err!=nil{t.Fatal(err)}
 return dir
}
func TestStoreActivateRollback(t *testing.T){
 base:=t.TempDir()
 store:=Store{Root:filepath.Join(base,"store")}
 v1:=fixture(t,base,"v1")
 v2:=fixture(t,base,"v2")
 if err:=store.Stage(v1,"v1");err!=nil{t.Fatal(err)}
 if err:=store.Stage(v2,"v2");err!=nil{t.Fatal(err)}
 if err:=store.Activate("v1");err!=nil{t.Fatal(err)}
 if err:=store.Activate("v2");err!=nil{t.Fatal(err)}
 current,err:=store.Current();if err!=nil||current!="v2"{t.Fatalf("current=%q err=%v",current,err)}
 prev,err:=store.Previous();if err!=nil||prev!="v1"{t.Fatalf("previous=%q err=%v",prev,err)}
 if err:=store.Rollback();err!=nil{t.Fatal(err)}
 current,err=store.Current();if err!=nil||current!="v1"{t.Fatalf("rollback current=%q err=%v",current,err)}
 if err:=store.Verify("v2");err!=nil{t.Fatal(err)}
}
func TestStoreRejectsInvalidVersion(t *testing.T){
 s:=Store{Root:filepath.Join(t.TempDir(),"store")}
 for _,name:=range []string{"../oops","/tmp/oops",".","..","a/b",""}{
  if err:=s.Activate(name);err==nil{t.Fatalf("accepted %q",name)}
 }
}
func TestStoreRejectsTamperedManifest(t *testing.T){
 base:=t.TempDir();store:=Store{Root:filepath.Join(base,"store")}
 source:=fixture(t,base,"v1")
 if err:=os.WriteFile(filepath.Join(source,"sysroot","libc.so"),[]byte("unexpected"),0644);err!=nil{t.Fatal(err)}
 if err:=store.Stage(source,"v1");err==nil{t.Fatal("accepted tampered tree")}
}
func TestStoreWillNotOverwrite(t *testing.T){
 base:=t.TempDir();store:=Store{Root:filepath.Join(base,"store")}
 one:=fixture(t,base,"v1")
 if err:=store.Stage(one,"v1");err!=nil{t.Fatal(err)}
 two:=fixture(t,base,"v1")
 if err:=store.Stage(two,"v1");err==nil{t.Fatal("overwrote immutable generation")}
}

func TestStoreRejectsAncestorStage(t *testing.T){
 base:=t.TempDir()
 s:=Store{Root:filepath.Join(base,"store")}
 if err:=s.Setup();err!=nil{t.Fatal(err)}
 // A move of base would recursively move the store into itself.
 m,err:=Inventory(base,"v1");if err!=nil{t.Fatal(err)}
 data,err:=json.Marshal(m);if err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(base,"glenv-manifest.json"),data,0644);err!=nil{t.Fatal(err)}
 if err:=s.Stage(base,"v1");err==nil{t.Fatal("accepted ancestor of version store")}
}

func TestStoreRejectsSymlinkRoot(t *testing.T){
 base:=t.TempDir()
 actual:=filepath.Join(base,"real")
 if err:=os.Mkdir(actual,0755);err!=nil{t.Fatal(err)}
 link:=filepath.Join(base,"link")
 if err:=os.Symlink(actual,link);err!=nil{t.Fatal(err)}
 s:=Store{Root:link}
 if err:=s.Setup();err==nil{t.Fatal("accepted symlink store root")}
}
func TestStoreRejectsSymlinkRelease(t *testing.T){
 base:=t.TempDir()
 s:=Store{Root:filepath.Join(base,"store")}
 if err:=s.Setup();err!=nil{t.Fatal(err)}
 outside:=fixture(t,base,"v1")
 if err:=os.Symlink(outside,filepath.Join(s.Root,"versions","v1"));err!=nil{t.Fatal(err)}
 if err:=s.Verify("v1");err==nil{t.Fatal("accepted symlinked release")}
}
func TestSelectorWillNotReplaceForeignFile(t *testing.T){
 base:=t.TempDir()
 s:=Store{Root:filepath.Join(base,"store")}
 src:=fixture(t,base,"v1")
 if err:=s.Stage(src,"v1");err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(s.Root,"current"),[]byte("foreign"),0600);err!=nil{t.Fatal(err)}
 if err:=s.Activate("v1");err==nil{t.Fatal("overwrote foreign selector file")}
}

func TestStoreRejectsSymlinkVersionsDir(t *testing.T){
 base:=t.TempDir()
 root:=filepath.Join(base,"store")
 if err:=os.Mkdir(root,0755);err!=nil{t.Fatal(err)}
 external:=filepath.Join(base,"external")
 if err:=os.Mkdir(external,0755);err!=nil{t.Fatal(err)}
 if err:=os.Symlink(external,filepath.Join(root,"versions"));err!=nil{t.Fatal(err)}
 s:=Store{Root:root}
 if err:=s.Setup();err==nil{t.Fatal("accepted symlink versions directory")}
 src:=fixture(t,base,"v1")
 if err:=s.Stage(src,"v1");err==nil{t.Fatal("staged through symlinked versions directory")}
 if err:=s.Verify("v1");err==nil{t.Fatal("verified through symlinked versions directory")}
 if _,err:=os.Stat(filepath.Join(external,"v1"));!os.IsNotExist(err){t.Fatalf("wrote outside store: %v",err)}
}

func TestStoreRejectsSymlinkManifest(t *testing.T){
 base:=t.TempDir()
 source:=fixture(t,base,"v1")
 if err:=os.Remove(filepath.Join(source,"glenv-manifest.json"));err!=nil{t.Fatal(err)}
 external:=filepath.Join(base,"foreign-manifest.json")
 if err:=os.WriteFile(external,[]byte("{}"),0600);err!=nil{t.Fatal(err)}
 if err:=os.Symlink(external,filepath.Join(source,"glenv-manifest.json"));err!=nil{t.Fatal(err)}
 s:=Store{Root:filepath.Join(base,"store")}
 if err:=s.Stage(source,"v1");err==nil{t.Fatal("accepted external symlink manifest")}
}

func TestStageRefusesOriginalGlenvPrefix(t *testing.T){
 base:=t.TempDir()
 original:=fixture(t,base,"v1")
 if err:=os.WriteFile(filepath.Join(original,".glenv-prefix"),[]byte("original"),0600);err!=nil{t.Fatal(err)}
 s:=Store{Root:filepath.Join(base,"store")}
 if err:=s.Stage(original,"v1");err==nil{t.Fatal("stage moved an original glenv-owned prefix")}
 if _,err:=os.Stat(filepath.Join(original,"sysroot","libc.so"));err!=nil{
  t.Fatalf("original glenv was moved: %v",err)
 }
}
