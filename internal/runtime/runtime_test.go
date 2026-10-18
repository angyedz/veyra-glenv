package runtime

import (
 "errors"
 "os"
 "path/filepath"
 "testing"
)

func TestDetect(t *testing.T) {
 root := t.TempDir()
 if err := os.MkdirAll(filepath.Join(root,"lib64"),0755); err != nil { t.Fatal(err) }
 if _,err := Detect(root); !errors.Is(err,ErrNotFound) { t.Fatalf("expected ErrNotFound, got %v",err) }
 loader := filepath.Join(root,"lib64","ld-linux-x86-64.so.2")
 if err := os.WriteFile(loader,[]byte("test"),0644); err != nil { t.Fatal(err) }
 got,err := Detect(root)
 if err != nil { t.Fatal(err) }
 if got.Loader != loader { t.Fatalf("unexpected loader %q",got.Loader) }
}
