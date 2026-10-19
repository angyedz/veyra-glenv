package deployment

import (
 "crypto/rand"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "regexp"
 "strings"
 "syscall"
)

var releaseName=regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// Store holds pre-staged immutable generations. A higher-level, privileged
// boot coordinator is responsible for publishing the selected generation to
// both Veyra mount namespaces and for pairing it with the host deployment.
type Store struct{ Root string }

func (s Store) release(version string)(string,error){
 if !releaseName.MatchString(version)||version=="."||version==".." {
  return "",fmt.Errorf("invalid release identifier %q",version)
 }
 return filepath.Join(s.Root,"versions",version),nil
}
// verifyStoreRoot prevents untrusted symlinks in the store's final path.
// The privileged boot coordinator must additionally own the parent directory
// and enforce access control; this check is not a substitute for that policy.
func (s Store) verifyStoreRoot()error{
 if s.Root==""||!filepath.IsAbs(s.Root){return errors.New("store root must be absolute")}
 info,err:=os.Lstat(s.Root)
 if os.IsNotExist(err){return nil}
 if err!=nil{return err}
 if !info.IsDir()||info.Mode()&os.ModeSymlink!=0{return errors.New("store root must be a real directory")}
 return nil
}
func (s Store) verifyVersionsDir()error{
 info,err:=os.Lstat(filepath.Join(s.Root,"versions"))
 if err!=nil{return err}
 if !info.IsDir()||info.Mode()&os.ModeSymlink!=0{
  return errors.New("versions directory must be a real directory")
 }
 return nil
}
func (s Store) Setup()error{
 if err:=s.verifyStoreRoot();err!=nil{return err}
 if err:=os.MkdirAll(s.Root,0755);err!=nil{return err}
 versions:=filepath.Join(s.Root,"versions")
 if err:=os.Mkdir(versions,0755);err!=nil&&!os.IsExist(err){return err}
 return s.verifyVersionsDir()
}
func readTree(dir string)(TreeManifest,error){
 var m TreeManifest
 manifest:=filepath.Join(dir,"glenv-manifest.json")
 info,err:=os.Lstat(manifest);if err!=nil{return m,err}
 if !info.Mode().IsRegular(){return m,errors.New("glenv manifest must be a regular file")}
 data,err:=os.ReadFile(manifest);if err!=nil{return m,err}
 if err=json.Unmarshal(data,&m);err!=nil{return m,err}
 return m,VerifyTree(dir,m)
}
func (s Store) Verify(version string)error{
 if err:=s.verifyStoreRoot();err!=nil{return err}
 if err:=s.verifyVersionsDir();err!=nil{return err}
 dir,err:=s.release(version);if err!=nil{return err}
 info,err:=os.Lstat(dir);if err!=nil{return err}
 if !info.IsDir()||info.Mode()&os.ModeSymlink!=0{return errors.New("release must be a real directory")}
 m,err:=readTree(dir);if err!=nil{return err}
 if m.Version!=version{return fmt.Errorf("release %s has mismatched manifest version %q",version,m.Version)}
 return nil
}

// Stage moves an already fully populated directory into the version store.
// It requires the source to be on the same filesystem (atomic rename) and
// never overwrites a release. Caller must exclusively own the source and
// prevent concurrent modifications; Stage cannot provide that protection.
func (s Store) Stage(source,version string)error{
 if err:=s.Setup();err!=nil{return err}
 dest,err:=s.release(version);if err!=nil{return err}
 source,err=filepath.Abs(source);if err!=nil{return err}
 // Never rename the installed original glenv prefix into the version store.
 // Original glenv creates .glenv-prefix as its installation ownership marker.
 if source=="/opt/glibc-env"||source=="/opt/glibc-env/sysroot"{
  return errors.New("refusing to move the installed glenv prefix; use release prepare-copy")
 }
 if _,err:=os.Lstat(filepath.Join(source,".glenv-prefix"));err==nil{
  return errors.New("refusing to move original glenv-owned prefix; use release prepare-copy")
 }else if !os.IsNotExist(err){return err}
 m,err:=readTree(source);if err!=nil{return err}
 if (m.Format!=2&&m.Format!=3)||m.Version!=version{return errors.New("manifest format or version mismatch")}
 if _,err:=os.Lstat(dest);err==nil{return os.ErrExist}else if !os.IsNotExist(err){return err}
 srcInfo,err:=os.Lstat(source);if err!=nil{return err}
 if !srcInfo.IsDir()||srcInfo.Mode()&os.ModeSymlink!=0{return errors.New("source is not a directory")}
 // Staging must never adopt a parent or child of the store tree.
 absStore,err:=filepath.Abs(s.Root);if err!=nil{return err}
 rel,err:=filepath.Rel(absStore,source);if err!=nil{return err}
 if rel=="."||(!strings.HasPrefix(rel,".."+string(filepath.Separator))&&rel!=".."){
  return errors.New("source is inside the version store")
 }
 // Also reject moving an ancestor of the store into itself.
 reverse,err:=filepath.Rel(source,absStore);if err!=nil{return err}
 if reverse=="."||(!strings.HasPrefix(reverse,".."+string(filepath.Separator))&&reverse!=".."){
  return errors.New("source contains the version store")
 }
 if err:=os.Rename(source,dest);err!=nil{return err}
 if err:=s.Verify(version);err!=nil{
  // Do not remove or roll back a moved tree blindly: report the staged
  // invalid generation for manual inspection.
  return fmt.Errorf("staged release failed verification: %w",err)
 }
 return syncDir(filepath.Join(s.Root,"versions"))
}

func (s Store) selected(link string)(string,error){
 if err:=s.verifyStoreRoot();err!=nil{return "",err}
 target,err:=os.Readlink(filepath.Join(s.Root,link))
 if os.IsNotExist(err){return "",nil}
 if err!=nil{return "",err}
 const prefix="versions/"
 if !strings.HasPrefix(target,prefix){return "",errors.New("unexpected selection target")}
 version:=strings.TrimPrefix(target,prefix)
 if _,err:=s.release(version);err!=nil{return "",err}
 if target!=filepath.ToSlash(filepath.Join("versions",version)){return "",errors.New("invalid selection target")}
 return version,nil
}
func (s Store) Current()(string,error){return s.selected("current")}
func (s Store) Previous()(string,error){return s.selected("previous")}

func randomSuffix()(string,error){
 var b [8]byte
 if _,err:=rand.Read(b[:]);err!=nil{return "",err}
 return hex.EncodeToString(b[:]),nil
}
func syncDir(p string)error{
 f,err:=os.Open(p);if err!=nil{return err};defer f.Close()
 err=f.Sync()
 if errors.Is(err,syscall.EINVAL){return nil}
 return err
}
func (s Store) replaceLink(link,version string)error{
 if _,err:=s.release(version);err!=nil{return err}
 destination:=filepath.Join(s.Root,link)
 if info,err:=os.Lstat(destination);err==nil {
  if info.Mode()&os.ModeSymlink==0{return fmt.Errorf("refusing to replace non-symlink selector: %s",destination)}
 }else if !os.IsNotExist(err){return err}
 suffix,err:=randomSuffix();if err!=nil{return err}
 temp:=filepath.Join(s.Root,".glenv-"+link+"-"+suffix)
 if err:=os.Symlink(filepath.Join("versions",version),temp);err!=nil{return err}
 defer os.Remove(temp)
 if err:=os.Rename(temp,filepath.Join(s.Root,link));err!=nil{return err}
 return syncDir(s.Root)
}

// Activate replaces the selection symlink atomically. This does not itself
// remount running processes, change Veyra's boot deployment, or implement
// power-loss-atomic pairing of current and previous pointers.
func (s Store) Activate(version string)error{
 if err:=s.Setup();err!=nil{return err}
 if err:=s.Verify(version);err!=nil{return err}
 current,err:=s.Current();if err!=nil{return err}
 if current==version{return nil}
 if current!="" {
  if err:=s.Verify(current);err!=nil{return fmt.Errorf("current generation invalid: %w",err)}
  if err:=s.replaceLink("previous",current);err!=nil{return err}
 }
 return s.replaceLink("current",version)
}
func (s Store) Rollback()error{
 previous,err:=s.Previous();if err!=nil{return err}
 if previous==""{return errors.New("no previous generation")}
 return s.Activate(previous)
}
