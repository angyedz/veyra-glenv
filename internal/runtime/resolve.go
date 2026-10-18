package runtime

import (
 "errors"
 "fmt"
 "os"
 "path"
 "path/filepath"
 "strings"
)

// resolveInSysroot interprets Debian absolute symlinks within the sysroot.
func resolveInSysroot(sysroot, relative string)(string,error){
 if filepath.IsAbs(relative){return "",errors.New("expected relative path")}
 root,err:=filepath.Abs(sysroot);if err!=nil{return "",err}
 root,err=filepath.EvalSymlinks(root);if err!=nil{return "",err}
 pending:=strings.Split(filepath.ToSlash(relative),"/")
 var parts []string
 links:=0
 for len(pending)>0{
  part:=pending[0];pending=pending[1:]
  if part==""||part=="."{continue}
  if part==".."{if len(parts)==0{return "",errors.New("path escapes sysroot")};parts=parts[:len(parts)-1];continue}
  parts=append(parts,part)
  candidate:=filepath.Join(append([]string{root},parts...)...)
  info,e:=os.Lstat(candidate);if e!=nil{return "",e}
  if info.Mode()&os.ModeSymlink==0{continue}
  links++;if links>40{return "",errors.New("too many symlinks")}
  target,e:=os.Readlink(candidate);if e!=nil{return "",e}
  parts=parts[:len(parts)-1]
  if path.IsAbs(target){parts=nil}
  pending=append(strings.Split(filepath.ToSlash(target),"/"),pending...)
 }
 result:=filepath.Join(append([]string{root},parts...)...)
 if !within(root,result){return "",fmt.Errorf("outside sysroot: %s",relative)}
 return result,nil
}
