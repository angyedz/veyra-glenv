package deployment

import (
 "os"
 "path/filepath"
 "testing"
)
func TestPinKeepsOldGenerationUntilNextBoot(t *testing.T){
 base:=t.TempDir()
 s:=Store{Root:filepath.Join(base,"store")}
 for _,v:=range []string{"v1","v2"}{
  source:=fixture(t,base,v)
  if err:=s.Stage(source,v);err!=nil{t.Fatal(err)}
 }
 if err:=s.Activate("v1");err!=nil{t.Fatal(err)}
 pin:=filepath.Join(base,"boot","runtime")
 if err:=s.Pin(pin);err!=nil{t.Fatal(err)}
 old,err:=os.Readlink(pin);if err!=nil{t.Fatal(err)}
 if filepath.Base(old)!="v1"{t.Fatalf("unexpected boot pin: %s",old)}
 if err:=s.Activate("v2");err!=nil{t.Fatal(err)}
 still,err:=os.Readlink(pin);if err!=nil{t.Fatal(err)}
 if still!=old{t.Fatal("live pin switched during activation")}
 if err:=s.Pin(pin);err!=nil{t.Fatal(err)}
 now,err:=os.Readlink(pin);if err!=nil{t.Fatal(err)}
 if filepath.Base(now)!="v2"{t.Fatalf("new boot pin not selected: %s",now)}
}
func TestPinWillNotOverwriteForeignFile(t *testing.T){
 base:=t.TempDir()
 s:=Store{Root:filepath.Join(base,"store")}
 source:=fixture(t,base,"v1")
 if err:=s.Stage(source,"v1");err!=nil{t.Fatal(err)}
 if err:=s.Activate("v1");err!=nil{t.Fatal(err)}
 path:=filepath.Join(base,"foreign")
 if err:=os.WriteFile(path,[]byte("keep"),0600);err!=nil{t.Fatal(err)}
 if err:=s.Pin(path);err==nil{t.Fatal("overwrote foreign file")}
}
