package deployment

import (
 "crypto/ed25519"
 "encoding/base64"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"
)

// Signature verification is intentionally separate from regular tree
// verification. A checksum manifest protects integrity only if its exact
// serialized bytes are authenticated with a trusted out-of-band public key.
// Keys must be pinned by the Veyra host deployment, never read from the
// untrusted glenv release being checked.
func ReadPublicKey(file string)(ed25519.PublicKey,error){
 data,err:=os.ReadFile(file);if err!=nil{return nil,err}
 key,err:=base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(string(data)))
 if err!=nil{return nil,fmt.Errorf("invalid base64 public key: %w",err)}
 if len(key)!=ed25519.PublicKeySize{return nil,errors.New("invalid Ed25519 public key size")}
 return ed25519.PublicKey(key),nil
}

// VerifySigned checks a detached Ed25519 signature over the *exact* manifest
// file bytes, and then verifies the complete version-3 release tree.
// Unlike Verify, it rejects legacy manifests without permission protection.
func VerifySigned(root string, publicKey ed25519.PublicKey) (TreeManifest,error){
 var manifest TreeManifest
 if len(publicKey)!=ed25519.PublicKeySize{return manifest,errors.New("invalid Ed25519 public key size")}
 info,err:=os.Lstat(root);if err!=nil{return manifest,err}
 if !info.IsDir()||info.Mode()&os.ModeSymlink!=0{return manifest,errors.New("release root must be a real directory")}
 manifestFile:=filepath.Join(root,"glenv-manifest.json")
 signatureFile:=filepath.Join(root,"glenv-manifest.json.sig")
 for _,name:=range []string{manifestFile,signatureFile}{
  info,err:=os.Lstat(name);if err!=nil{return manifest,err}
  if !info.Mode().IsRegular(){return manifest,fmt.Errorf("signed release metadata is not a regular file: %s",name)}
 }
 payload,err:=os.ReadFile(manifestFile);if err!=nil{return manifest,err}
 encoded,err:=os.ReadFile(signatureFile);if err!=nil{return manifest,err}
 signature,err:=base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(string(encoded)))
 if err!=nil{return manifest,fmt.Errorf("invalid release signature encoding: %w",err)}
 if len(signature)!=ed25519.SignatureSize||!ed25519.Verify(publicKey,payload,signature){
  return manifest,errors.New("glenv release signature verification failed")
 }
 if err:=json.Unmarshal(payload,&manifest);err!=nil{return manifest,err}
 if manifest.Format!=3||manifest.Version==""{return manifest,errors.New("signed releases require format-3 manifest")}
 if err:=VerifyTree(root,manifest);err!=nil{return manifest,err}
 return manifest,nil
}
