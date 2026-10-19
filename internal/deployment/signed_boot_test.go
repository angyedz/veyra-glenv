package deployment

import (
 "crypto/ed25519"
 "crypto/rand"
 "os"
 "path/filepath"
 "testing"
)

func TestSignedBootPinRejectsUnsignedAndWrongKey(t *testing.T){
 base:=t.TempDir()
 s:=Store{Root:filepath.Join(base,"store")}
 unsigned:=fixture(t,base,"unsigned")
 if err:=s.Stage(unsigned,"unsigned");err!=nil{t.Fatal(err)}
 if err:=s.PairHost("hostA","unsigned");err!=nil{t.Fatal(err)}
 pub,_,err:=ed25519.GenerateKey(rand.Reader);if err!=nil{t.Fatal(err)}
 pin:=filepath.Join(base,"boot","runtime")
 if err:=s.PinPairedHostSigned("hostA",pin,pub);err==nil{t.Fatal("pinned unsigned generation")}
 if _,err:=os.Lstat(pin);!os.IsNotExist(err){t.Fatalf("unexpected pin: %v",err)}
}

func TestSignedBootPinPairedVersion(t *testing.T){
 base:=t.TempDir()
 root,pub:=signedFixture(t)
 s:=Store{Root:filepath.Join(base,"store")}
 if err:=s.Stage(root,"v1");err!=nil{t.Fatal(err)}
 if err:=s.PairHost("hostA","v1");err!=nil{t.Fatal(err)}
 pin:=filepath.Join(base,"boot","runtime")
 wrong,_,err:=ed25519.GenerateKey(rand.Reader);if err!=nil{t.Fatal(err)}
 if err:=s.PinPairedHostSigned("hostA",pin,wrong);err==nil{t.Fatal("wrong trusted key accepted")}
 if err:=s.PinPairedHostSigned("hostA",pin,pub);err!=nil{t.Fatal(err)}
 target,err:=os.Readlink(pin);if err!=nil{t.Fatal(err)}
 if filepath.Base(target)!="v1"{t.Fatalf("wrong generation: %s",target)}
 if err:=s.PinPairedHostSigned("otherHost",pin,pub);err==nil{t.Fatal("unknown host accepted")}
}
