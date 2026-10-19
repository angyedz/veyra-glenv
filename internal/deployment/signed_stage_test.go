package deployment

import (
 "crypto/ed25519"
 "crypto/rand"
 "os"
 "path/filepath"
 "testing"
)

func TestStageSignedAcceptsVerifiedRelease(t *testing.T){
 base:=t.TempDir()
 source,key:=signedFixture(t)
 s:=Store{Root:filepath.Join(base,"store")}
 if err:=s.StageSigned(source,"v1",key);err!=nil{t.Fatal(err)}
 if err:=s.VerifySignedVersion("v1",key);err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(source);!os.IsNotExist(err){t.Fatalf("staged source still present: %v",err)}
}
func TestStageSignedRejectsWrongKeyWithoutMovingSource(t *testing.T){
 s:=Store{Root:filepath.Join(t.TempDir(),"store")}
 source,_:=signedFixture(t)
 other,_,err:=ed25519.GenerateKey(rand.Reader);if err!=nil{t.Fatal(err)}
 if err:=s.StageSigned(source,"v1",other);err==nil{t.Fatal("accepted foreign signer")}
 if _,err:=os.Stat(filepath.Join(source,"glenv-manifest.json"));err!=nil{t.Fatalf("source changed: %v",err)}
}
func TestStageSignedRejectsUnsignedAndVersionMismatch(t *testing.T){
 base:=t.TempDir()
 s:=Store{Root:filepath.Join(base,"store")}
 unsigned:=fixture(t,base,"unsigned")
 key,_,err:=ed25519.GenerateKey(rand.Reader);if err!=nil{t.Fatal(err)}
 if err:=s.StageSigned(unsigned,"unsigned",key);err==nil{t.Fatal("accepted unsigned source")}
 if _,err:=os.Stat(unsigned);err!=nil{t.Fatalf("unsigned source removed: %v",err)}
 source,pub:=signedFixture(t)
 if err:=s.StageSigned(source,"unexpected-version",pub);err==nil{t.Fatal("accepted signed manifest version mismatch")}
 if _,err:=os.Stat(source);err!=nil{t.Fatalf("mismatched source moved: %v",err)}
}
