package deployment

import (
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"
)

// Pin creates a boot-session-specific symlink directly to the selected,
// verified generation. Unlike 'current', the pin never follows future
// activation changes. A Veyra boot coordinator must mount or expose this
// pinned release to host and user namespaces and keep its files immutable.
// This method does not mount, unmount, or change namespace privileges.
func (s Store) Pin(destination string)error{
 current,err:=s.Current();if err!=nil{return err}
 if current==""{return errors.New("no active glenv generation")}
 return s.pinVersion(current,destination)
}

// pinVersion never changes the global active selector. BindHost/PinForHost
// can use a deployment-specific version instead of whichever is current.
func (s Store) pinVersion(version,destination string)error{
 if !filepath.IsAbs(destination)||filepath.Clean(destination)=="/"{
  return errors.New("pin destination must be an absolute non-root path")
 }
 if err:=s.Verify(version);err!=nil{return err}
 release,err:=s.release(version);if err!=nil{return err}
 parent:=filepath.Dir(destination)
 if err:=os.MkdirAll(parent,0755);err!=nil{return err}
 // A pin may only replace a previous pin, never a directory or a
 // non-symlink object installed by another service.
 if info,err:=os.Lstat(destination);err==nil{
  if info.Mode()&os.ModeSymlink==0{return fmt.Errorf("pin destination exists and is not a symlink: %s",destination)}
  old,err:=os.Readlink(destination);if err!=nil{return err}
  if !strings.HasPrefix(old,filepath.Join(s.Root,"versions")+string(filepath.Separator)){return fmt.Errorf("refusing to overwrite foreign symlink: %s",destination)}
 }else if !os.IsNotExist(err){return err}
 suffix,err:=randomSuffix();if err!=nil{return err}
 temp:=filepath.Join(parent,".glenv-pin-"+suffix)
 if err:=os.Symlink(release,temp);err!=nil{return err}
 defer os.Remove(temp)
 if err:=os.Rename(temp,destination);err!=nil{return err}
 return syncDir(parent)
}
