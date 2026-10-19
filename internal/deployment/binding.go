package deployment

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
)

// BootBinding pairs exactly one Veyra host deployment with one glenv release.
// It is local selection metadata, NOT a cryptographic authenticity statement.
type BootBinding struct {
 Format int `json:"format"`
 HostDeployment string `json:"host_deployment"`
 GlenvVersion string `json:"glenv_version"`
}

const bindingName = "boot-binding.json"

// ReadBinding reads the single boot record. Unlike two separate selection
// pointers, its two identifiers are always read from one filesystem object.
func (s Store) ReadBinding()(BootBinding,error){
 var b BootBinding
 if err:=s.verifyStoreRoot();err!=nil{return b,err}
 filename:=filepath.Join(s.Root,bindingName)
 info,err:=os.Lstat(filename);if err!=nil{return b,err}
 if !info.Mode().IsRegular(){return b,errors.New("boot binding must be a regular file")}
 data,err:=os.ReadFile(filename);if err!=nil{return b,err}
 if err:=json.Unmarshal(data,&b);err!=nil{return b,err}
 if b.Format!=1||!releaseName.MatchString(b.HostDeployment)||!releaseName.MatchString(b.GlenvVersion){
  return b,errors.New("invalid boot binding")
 }
 return b,nil
}

// BindHost publishes one crash-consistent pair of Veyra deployment and glenv
// generation IDs. Only trusted update management with an exclusive store lock
// may call this: it does NOT verify the Veyra deployment or its signatures.
func (s Store) BindHost(hostDeployment,glenvVersion string)error{
 if !releaseName.MatchString(hostDeployment){return errors.New("invalid Veyra deployment identifier")}
 if _,err:=s.release(glenvVersion);err!=nil{return err}
 if err:=s.Setup();err!=nil{return err}
 if err:=s.Verify(glenvVersion);err!=nil{return err}
 dest:=filepath.Join(s.Root,bindingName)
 if info,err:=os.Lstat(dest);err==nil {
  if !info.Mode().IsRegular(){return fmt.Errorf("refusing to replace non-regular boot binding: %s",dest)}
 }else if !os.IsNotExist(err){return err}
 b:=BootBinding{Format:1,HostDeployment:hostDeployment,GlenvVersion:glenvVersion}
 payload,err:=json.MarshalIndent(b,"","  ");if err!=nil{return err}
 payload=append(payload,'\n')
 suffix,err:=randomSuffix();if err!=nil{return err}
 temp:=filepath.Join(s.Root,".boot-binding-"+suffix)
 f,err:=os.OpenFile(temp,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600);if err!=nil{return err}
 defer os.Remove(temp)
 if _,err=f.Write(payload);err!=nil{f.Close();return err}
 if err=f.Sync();err!=nil{f.Close();return err}
 if err=f.Close();err!=nil{return err}
 if err=os.Rename(temp,dest);err!=nil{return err}
 return syncDir(s.Root)
}

// PinForHost refuses to boot an unrelated runtime if the bootloader picked
// another Veyra deployment. The coordinator must call this *before* starting
// either the host's glibc applications or the user environment.
func (s Store) PinForHost(hostDeployment,destination string)error{
 if !releaseName.MatchString(hostDeployment){return errors.New("invalid Veyra deployment identifier")}
 b,err:=s.ReadBinding();if err!=nil{return err}
 if b.HostDeployment!=hostDeployment {
  return fmt.Errorf("boot deployment mismatch: selected %s, binding expects %s",hostDeployment,b.HostDeployment)
 }
 return s.pinVersion(b.GlenvVersion,destination)
}
