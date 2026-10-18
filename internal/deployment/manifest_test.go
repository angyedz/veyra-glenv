package deployment

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "os"
 "path/filepath"
 "testing"
)
func TestVerify(t *testing.T){
 root:=t.TempDir()
 payload:=[]byte("wayland shim placeholder")
 if err:=os.WriteFile(filepath.Join(root,"shim.so"),payload,0644);err!=nil{t.Fatal(err)}
 sum:=sha256.Sum256(payload)
 m:=Manifest{Format:1,Version:"test-v1",Files:[]File{{Path:"shim.so",SHA256:hex.EncodeToString(sum[:])}}}
 data,err:=json.Marshal(m);if err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(root,"glenv-manifest.json"),data,0644);err!=nil{t.Fatal(err)}
 if _,err:=Verify(root);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(root,"shim.so"),[]byte("changed"),0644);err!=nil{t.Fatal(err)}
 if _,err:=Verify(root);err==nil{t.Fatal("accepted tampered release")}
}
func TestRejectManifestTraversal(t *testing.T){
 root:=t.TempDir()
 m:=Manifest{Format:1,Version:"test",Files:[]File{{Path:"../outside",SHA256:"0000000000000000000000000000000000000000000000000000000000000000"}}}
 data,_:=json.Marshal(m)
 if err:=os.WriteFile(filepath.Join(root,"glenv-manifest.json"),data,0644);err!=nil{t.Fatal(err)}
 if _,err:=Verify(root);err==nil{t.Fatal("accepted traversal")}
}
