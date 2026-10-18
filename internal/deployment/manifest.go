// Package deployment validates immutable glenv runtime releases before activation.
package deployment

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "strings"
)

type File struct {
 Path string `json:"path"`
 SHA256 string `json:"sha256"`
}
type Manifest struct {
 Format int `json:"format"`
 Version string `json:"version"`
 Files []File `json:"files"`
}

// Verify validates a complete, already-staged runtime. It never modifies the
// runtime or selects an active deployment.
func Verify(root string) (Manifest,error) {
 var m Manifest
 abs,err:=filepath.Abs(root);if err!=nil{return m,err}
 base,err:=filepath.EvalSymlinks(abs);if err!=nil{return m,err}
 data,err:=os.ReadFile(filepath.Join(base,"glenv-manifest.json"));if err!=nil{return m,err}
 if err=json.Unmarshal(data,&m);err!=nil{return m,err}
 if m.Format!=1||m.Version==""||len(m.Files)==0{return m,errors.New("invalid glenv manifest")}
 seen:=map[string]bool{}
 for _,f:=range m.Files{
  if f.Path==""||filepath.IsAbs(f.Path)||filepath.Clean(f.Path)!=f.Path||f.Path==".."||strings.HasPrefix(f.Path,".."+string(filepath.Separator))||len(f.SHA256)!=64||seen[f.Path]{
   return m,fmt.Errorf("invalid manifest entry: %q",f.Path)
  }
  seen[f.Path]=true
  expected,err:=hex.DecodeString(f.SHA256);if err!=nil{return m,err}
  name:=filepath.Join(base,f.Path)
  info,err:=os.Lstat(name);if err!=nil{return m,err}
  // Symlinks are not accepted in the initial manifest format: a later
  // format must explicitly describe and validate their targets.
  if !info.Mode().IsRegular(){return m,fmt.Errorf("not a regular file: %s",f.Path)}
  file,err:=os.Open(name);if err!=nil{return m,err}
  h:=sha256.New()
  _,copyErr:=io.Copy(h,file);closeErr:=file.Close()
  if copyErr!=nil{return m,copyErr};if closeErr!=nil{return m,closeErr}
  if !equal(h.Sum(nil),expected){return m,fmt.Errorf("checksum mismatch: %s",f.Path)}
 }
 return m,nil
}
func equal(a,b []byte)bool{
 if len(a)!=len(b){return false}
 var v byte;for i:=range a{v|=a[i]^b[i]};return v==0
}
