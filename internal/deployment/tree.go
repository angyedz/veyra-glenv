package deployment

import (
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "fmt"
 "io"
 "os"
 "path"
 "path/filepath"
 "sort"
 "strings"
)

// Entry describes one node of a complete runtime tree.
// Paths are slash-separated and relative to the release root.
type Entry struct {
 Path string `json:"path"`
 Kind string `json:"kind"` // file, dir, symlink
 SHA256 string `json:"sha256,omitempty"`
 Target string `json:"target,omitempty"`
 Mode uint32 `json:"mode,omitempty"` // Unix permission and special bits (format 3)
}

// TreeManifest is the strict, complete manifest format for a staged release.
type TreeManifest struct {
 Format int `json:"format"`
 Version string `json:"version"`
 Entries []Entry `json:"entries"`
}

func validPath(p string) bool {
 return p!="" && p!="." && !strings.HasPrefix(p,"/") && path.Clean(p)==p && !strings.HasPrefix(p,"../") && p!=".." && !strings.Contains(p,"\\")
}

func permissionBits(mode os.FileMode)uint32{
 bits:=uint32(mode.Perm())
 if mode&os.ModeSetuid!=0{bits|=04000}
 if mode&os.ModeSetgid!=0{bits|=02000}
 if mode&os.ModeSticky!=0{bits|=01000}
 return bits
}

func checksum(file string)(string,error){
 f,err:=os.Open(file);if err!=nil{return "",err};defer f.Close()
 h:=sha256.New();if _,err=io.Copy(h,f);err!=nil{return "",err}
 return hex.EncodeToString(h.Sum(nil)),nil
}

// Inventory walks a staged release without following symlinks.
// The manifest file itself is excluded to avoid self-referential hashing.
func Inventory(root,version string)(TreeManifest,error){
 m:=TreeManifest{Format:3,Version:version}
 if version==""{return m,errors.New("empty version")}
 base,err:=filepath.Abs(root);if err!=nil{return m,err}
 err=filepath.WalkDir(base,func(name string,d os.DirEntry,walkErr error)error{
  if walkErr!=nil{return walkErr}
  if name==base{return nil}
  rel,err:=filepath.Rel(base,name);if err!=nil{return err}
  rel=filepath.ToSlash(rel)
  if rel=="glenv-manifest.json"||rel=="glenv-manifest.json.sig"{return nil}
  if !validPath(rel){return fmt.Errorf("invalid release path: %q",rel)}
  entry:=Entry{Path:rel}
  info,err:=os.Lstat(name);if err!=nil{return err}
  switch {
  case d.IsDir():entry.Kind="dir";entry.Mode=permissionBits(info.Mode())
  case d.Type()&os.ModeSymlink!=0:
   entry.Kind="symlink"
   entry.Target,err=os.Readlink(name);if err!=nil{return err}
  case d.Type().IsRegular():
   entry.Kind="file"
   entry.Mode=permissionBits(info.Mode())
   entry.SHA256,err=checksum(name);if err!=nil{return err}
  default:return fmt.Errorf("unsupported file type: %s",rel)
  }
  m.Entries=append(m.Entries,entry)
  return nil
 })
 sort.Slice(m.Entries,func(i,j int)bool{return m.Entries[i].Path<m.Entries[j].Path})
 return m,err
}

// VerifyTree checks both content and completeness. It is read-only.
// It does not establish authenticity: a trusted signature is required before
// accepting manifests obtained from an untrusted source.
func VerifyTree(root string,expected TreeManifest)error{
 if (expected.Format!=2&&expected.Format!=3)||expected.Version==""{return errors.New("invalid tree manifest header")}
 actual,err:=Inventory(root,expected.Version);if err!=nil{return err}
 // Legacy format-2 manifests did not record permissions. Preserve the
 // ability to verify those releases, but format 3 is required for new ones.
 if expected.Format==2{for i:=range actual.Entries{actual.Entries[i].Mode=0}}
 if len(expected.Entries)!=len(actual.Entries){return fmt.Errorf("tree entry count mismatch: expected %d, found %d",len(expected.Entries),len(actual.Entries))}
 seen:=make(map[string]bool,len(expected.Entries))
 for _,e:=range expected.Entries{
  if !validPath(e.Path)||seen[e.Path]{return fmt.Errorf("invalid or duplicate entry: %q",e.Path)}
  seen[e.Path]=true
  if e.Kind!="file"&&e.Kind!="dir"&&e.Kind!="symlink"{return fmt.Errorf("invalid entry type: %s",e.Path)}
  if e.Kind=="file" {
   if len(e.SHA256)!=64{return fmt.Errorf("invalid digest: %s",e.Path)}
   if _,err:=hex.DecodeString(e.SHA256);err!=nil{return err}
  } else if e.SHA256!=""{return fmt.Errorf("unexpected digest: %s",e.Path)}
  if e.Kind=="symlink"&&e.Target==""{return fmt.Errorf("empty symlink target: %s",e.Path)}
  if e.Kind=="symlink"&&e.Mode!=0{return fmt.Errorf("invalid symlink permissions: %s",e.Path)}
  if expected.Format==2&&e.Mode!=0{return fmt.Errorf("legacy manifest unexpectedly sets mode: %s",e.Path)}
  if e.Mode &^ 07777 != 0{return fmt.Errorf("invalid permission bits: %s",e.Path)}
 }
 entries:=append([]Entry(nil),expected.Entries...)
 sort.Slice(entries,func(i,j int)bool{return entries[i].Path<entries[j].Path})
 for i,e:=range entries{
  a:=actual.Entries[i]
  if e!=a{return fmt.Errorf("tree mismatch at %s",e.Path)}
 }
 return nil
}
