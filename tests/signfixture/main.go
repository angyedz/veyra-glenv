// Command signfixture creates an EPHEMERAL, CI-ONLY Ed25519 test signature.
// It never writes a private key. Do not use its public key for production OTA.
package main

import (
 "crypto/ed25519"
 "crypto/rand"
 "encoding/base64"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"

 "github.com/angyedz/veyra-glenv/internal/deployment"
)

func main() {
 if err:=run(os.Args[1:]);err!=nil{
  fmt.Fprintln(os.Stderr,"signfixture:",err)
  os.Exit(1)
 }
}
func run(args []string) error {
 if len(args)!=2{return errors.New("usage: go run ./tests/signfixture <release-dir> <public-key-output>")}
 root,err:=filepath.Abs(args[0]);if err!=nil{return err}
 out,err:=filepath.Abs(args[1]);if err!=nil{return err}
 rel,err:=filepath.Rel(root,out);if err!=nil{return err}
 if rel=="."||(!strings.HasPrefix(rel,".."+string(filepath.Separator))&&rel!=".."){
  return errors.New("test public key must be outside the release tree")
 }
 manifestPath:=filepath.Join(root,"glenv-manifest.json")
 info,err:=os.Lstat(manifestPath);if err!=nil{return err}
 if !info.Mode().IsRegular(){return errors.New("manifest must be a regular file")}
 payload,err:=os.ReadFile(manifestPath);if err!=nil{return err}
 var manifest deployment.TreeManifest
 if err:=json.Unmarshal(payload,&manifest);err!=nil{return err}
 if manifest.Format!=3{return errors.New("test signing requires format-3 manifest")}
 if err:=deployment.VerifyTree(root,manifest);err!=nil{return err}
 public,private,err:=ed25519.GenerateKey(rand.Reader);if err!=nil{return err}
 signatureFile:=filepath.Join(root,"glenv-manifest.json.sig")
 sig,err:=os.OpenFile(signatureFile,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0644);if err!=nil{return err}
 encoded:=base64.StdEncoding.EncodeToString(ed25519.Sign(private,payload))
 if _,err=sig.WriteString(encoded+"\n");err!=nil{sig.Close();return err}
 if err:=sig.Close();err!=nil{return err}
 keyFile,err:=os.OpenFile(out,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0644);if err!=nil{return err}
 if _,err=keyFile.WriteString(base64.StdEncoding.EncodeToString(public)+"\n");err!=nil{keyFile.Close();return err}
 if err:=keyFile.Close();err!=nil{return err}
 if _,err:=deployment.VerifySigned(root,public);err!=nil{return err}
 fmt.Println("Ephemeral CI-only glenv release fixture signed")
 return nil
}
