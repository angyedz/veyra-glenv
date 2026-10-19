package deployment

import (
 "os"
 "path/filepath"
 "testing"
)

func TestBindAndPinHostDeployment(t *testing.T) {
 base:=t.TempDir()
 s:=Store{Root:filepath.Join(base,"store")}
 for _,version:=range []string{"v1","v2"}{
  if err:=s.Stage(fixture(t,base,version),version);err!=nil{t.Fatal(err)}
 }
 if err:=s.Activate("v2");err!=nil{t.Fatal(err)}
 if err:=s.BindHost("veyra-A","v1");err!=nil{t.Fatal(err)}
 got,err:=s.ReadBinding();if err!=nil{t.Fatal(err)}
 if got.HostDeployment!="veyra-A"||got.GlenvVersion!="v1"{t.Fatalf("unexpected boot pair: %+v",got)}
 pin:=filepath.Join(base,"boot","runtime")
 if err:=s.PinForHost("veyra-B",pin);err==nil{t.Fatal("mismatched host deployment permitted")}
 if _,err:=os.Lstat(pin);!os.IsNotExist(err){t.Fatalf("unexpected pin: %v",err)}
 if err:=s.PinForHost("veyra-A",pin);err!=nil{t.Fatal(err)}
 target,err:=os.Readlink(pin);if err!=nil{t.Fatal(err)}
 if filepath.Base(target)!="v1"{t.Fatalf("pinned current instead of host binding: %s",target)}
 if err:=s.BindHost("veyra-B","v2");err!=nil{t.Fatal(err)}
 // The active boot pin stays on v1 until a later explicit boot selection.
 still,err:=os.Readlink(pin);if err!=nil||still!=target{t.Fatalf("pin switched unexpectedly: %v %q",err,still)}
 if err:=s.PinForHost("veyra-B",pin);err!=nil{t.Fatal(err)}
 latest,err:=os.Readlink(pin);if err!=nil||filepath.Base(latest)!="v2"{t.Fatalf("latest: %v %q",err,latest)}
}

func TestBindingRejectsInvalidAndForeignFiles(t *testing.T){
 base:=t.TempDir()
 s:=Store{Root:filepath.Join(base,"store")}
 if err:=s.Stage(fixture(t,base,"v1"),"v1");err!=nil{t.Fatal(err)}
 if err:=s.BindHost("../bad","v1");err==nil{t.Fatal("invalid deployment accepted")}
 if err:=s.BindHost("veyra","missing");err==nil{t.Fatal("missing glenv version accepted")}
 foreign:=filepath.Join(base,"foreign")
 if err:=os.WriteFile(foreign,[]byte("external"),0600);err!=nil{t.Fatal(err)}
 if err:=os.Symlink(foreign,filepath.Join(s.Root,bindingName));err!=nil{t.Fatal(err)}
 if err:=s.BindHost("veyra","v1");err==nil{t.Fatal("overwrote foreign binding")}
 if _,err:=s.ReadBinding();err==nil{t.Fatal("accepted symlink binding")}
}
