package deployment

import (
 "crypto/ed25519"
 "crypto/rand"
 "encoding/base64"
 "encoding/json"
 "os"
 "path/filepath"
 "testing"
)

func signedFixture(t *testing.T)(string,ed25519.PublicKey){
 t.Helper()
 root:=t.TempDir()
 if err:=os.Mkdir(filepath.Join(root,"sysroot"),0755);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(root,"sysroot","libc.so"),[]byte("test libc"),0644);err!=nil{t.Fatal(err)}
 manifest,err:=Inventory(root,"v1");if err!=nil{t.Fatal(err)}
 payload,err:=json.Marshal(manifest);if err!=nil{t.Fatal(err)}
 public,private,err:=ed25519.GenerateKey(rand.Reader);if err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(root,"glenv-manifest.json"),payload,0644);err!=nil{t.Fatal(err)}
 sig:=base64.StdEncoding.EncodeToString(ed25519.Sign(private,payload))
 if err:=os.WriteFile(filepath.Join(root,"glenv-manifest.json.sig"),[]byte(sig+"\n"),0644);err!=nil{t.Fatal(err)}
 return root,public
}
func TestVerifySignedManifest(t *testing.T){
 root,pub:=signedFixture(t)
 if _,err:=VerifySigned(root,pub);err!=nil{t.Fatal(err)}
}
func TestVerifySignedRejectsModifiedTree(t *testing.T){
 root,pub:=signedFixture(t)
 if err:=os.WriteFile(filepath.Join(root,"sysroot","libc.so"),[]byte("tampered"),0644);err!=nil{t.Fatal(err)}
 if _,err:=VerifySigned(root,pub);err==nil{t.Fatal("accepted tampered tree")}
}
func TestVerifySignedRejectsTamperedManifest(t *testing.T){
 root,pub:=signedFixture(t)
 file:=filepath.Join(root,"glenv-manifest.json")
 data,err:=os.ReadFile(file);if err!=nil{t.Fatal(err)}
 data=append(data,' ')
 if err:=os.WriteFile(file,data,0644);err!=nil{t.Fatal(err)}
 if _,err:=VerifySigned(root,pub);err==nil{t.Fatal("accepted tampered manifest")}
}
func TestVerifySignedRejectsWrongKey(t *testing.T){
 root,_:=signedFixture(t)
 another,_,err:=ed25519.GenerateKey(rand.Reader);if err!=nil{t.Fatal(err)}
 if _,err:=VerifySigned(root,another);err==nil{t.Fatal("accepted unknown signer")}
}
func TestVerifySignedRejectsSignatureSymlink(t *testing.T){
 root,pub:=signedFixture(t)
 path:=filepath.Join(root,"glenv-manifest.json.sig")
 payload,err:=os.ReadFile(path);if err!=nil{t.Fatal(err)}
 if err:=os.Remove(path);err!=nil{t.Fatal(err)}
 target:=filepath.Join(t.TempDir(),"sig")
 if err:=os.WriteFile(target,payload,0644);err!=nil{t.Fatal(err)}
 if err:=os.Symlink(target,path);err!=nil{t.Fatal(err)}
 if _,err:=VerifySigned(root,pub);err==nil{t.Fatal("accepted symlinked signature")}
}
func TestReadPublicKey(t *testing.T){
 root,pub:=signedFixture(t)
 keyfile:=filepath.Join(t.TempDir(),"key.pub")
 if err:=os.WriteFile(keyfile,[]byte(base64.StdEncoding.EncodeToString(pub)+"\n"),0644);err!=nil{t.Fatal(err)}
 loaded,err:=ReadPublicKey(keyfile);if err!=nil{t.Fatal(err)}
 if string(loaded)!=string(pub){t.Fatal("wrong public key")}
 if err:=os.WriteFile(keyfile,[]byte("invalid"),0644);err!=nil{t.Fatal(err)}
 if _,err:=ReadPublicKey(keyfile);err==nil{t.Fatal("accepted invalid key")}
 if _,err:=VerifySigned(root,loaded);err!=nil{t.Fatal(err)}
}

func TestVerifySignedRejectsOversizedSignature(t *testing.T){
 root,pub:=signedFixture(t)
 huge:=make([]byte,maxSignatureBytes+1)
 for i:=range huge{huge[i]='A'}
 if err:=os.WriteFile(filepath.Join(root,"glenv-manifest.json.sig"),huge,0644);err!=nil{t.Fatal(err)}
 if _,err:=VerifySigned(root,pub);err==nil{t.Fatal("accepted oversized signature")}
}
func TestReadPublicKeyRejectsSymlinkAndOversize(t *testing.T){
 _,pub:=signedFixture(t)
 dir:=t.TempDir()
 real:=filepath.Join(dir,"real.pub")
 key:=filepath.Join(dir,"linked.pub")
 if err:=os.WriteFile(real,[]byte(base64.StdEncoding.EncodeToString(pub)),0644);err!=nil{t.Fatal(err)}
 if err:=os.Symlink(real,key);err!=nil{t.Fatal(err)}
 if _,err:=ReadPublicKey(key);err==nil{t.Fatal("accepted symlinked trust key")}
 oversize:=filepath.Join(dir,"huge.pub")
 if err:=os.WriteFile(oversize,make([]byte,maxTrustedPublicKeyBytes+1),0644);err!=nil{t.Fatal(err)}
 if _,err:=ReadPublicKey(oversize);err==nil{t.Fatal("accepted oversized trust key")}
}
