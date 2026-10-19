package deployment

import (
 "os"
 "path/filepath"
 "testing"
)
func makeTree(t *testing.T)string{
 t.Helper()
 root:=t.TempDir()
 if err:=os.MkdirAll(filepath.Join(root,"sysroot/usr/lib"),0755);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(root,"sysroot/usr/lib/libc.so.6"),[]byte("glibc fixture"),0644);err!=nil{t.Fatal(err)}
 if err:=os.Symlink("/usr/lib/libc.so.6",filepath.Join(root,"sysroot/libc.so.6"));err!=nil{t.Fatal(err)}
 return root
}
func TestTreeRoundTrip(t *testing.T){
 root:=makeTree(t)
 m,err:=Inventory(root,"release-1");if err!=nil{t.Fatal(err)}
 if err:=VerifyTree(root,m);err!=nil{t.Fatal(err)}
 if len(m.Entries)!=5{t.Fatalf("unexpected entry count: %d",len(m.Entries))}
}
func TestTreeDetectsMutation(t *testing.T){
 root:=makeTree(t)
 m,err:=Inventory(root,"release-1");if err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(root,"sysroot/usr/lib/libc.so.6"),[]byte("tampered"),0644);err!=nil{t.Fatal(err)}
 if err:=VerifyTree(root,m);err==nil{t.Fatal("accepted modified file")}
}
func TestTreeDetectsUnexpectedFile(t *testing.T){
 root:=makeTree(t)
 m,err:=Inventory(root,"release-1");if err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(root,"extra"),[]byte("unexpected"),0644);err!=nil{t.Fatal(err)}
 if err:=VerifyTree(root,m);err==nil{t.Fatal("accepted unlisted file")}
}
func TestTreeDetectsSymlinkChange(t *testing.T){
 root:=makeTree(t)
 m,err:=Inventory(root,"release-1");if err!=nil{t.Fatal(err)}
 link:=filepath.Join(root,"sysroot/libc.so.6")
 if err:=os.Remove(link);err!=nil{t.Fatal(err)}
 if err:=os.Symlink("/tmp/host-libc",link);err!=nil{t.Fatal(err)}
 if err:=VerifyTree(root,m);err==nil{t.Fatal("accepted changed symlink")}
}

func TestFormat3RejectsPermissionMutation(t *testing.T){
 root:=makeTree(t)
 manifest,err:=Inventory(root,"release-1");if err!=nil{t.Fatal(err)}
 if manifest.Format!=3{t.Fatalf("unexpected inventory format %d",manifest.Format)}
 binary:=filepath.Join(root,"sysroot/usr/lib/libc.so.6")
 if err:=os.Chmod(binary,0600);err!=nil{t.Fatal(err)}
 if err:=VerifyTree(root,manifest);err==nil{t.Fatal("accepted changed glenv file permissions")}
}
func TestLegacyFormat2CanStillVerify(t *testing.T){
 root:=makeTree(t)
 old,err:=Inventory(root,"legacy");if err!=nil{t.Fatal(err)}
 old.Format=2
 for i:=range old.Entries{old.Entries[i].Mode=0}
 if err:=VerifyTree(root,old);err!=nil{t.Fatalf("legacy format 2 compatibility: %v",err)}
}
