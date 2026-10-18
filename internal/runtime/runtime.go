package runtime

import (
 "errors"
 "fmt"
 "os"
 "path/filepath"
)

var ErrNotFound = errors.New("glenv runtime not found")

type Runtime struct {
 Root string
 Loader string
}

func Detect(root string) (Runtime, error) {
 if root == "" { return Runtime{}, ErrNotFound }
 absolute, err := filepath.Abs(root)
 if err != nil { return Runtime{}, err }
 info, err := os.Stat(absolute)
 if err != nil { if os.IsNotExist(err) { return Runtime{}, ErrNotFound }; return Runtime{}, err }
 if !info.IsDir() { return Runtime{}, fmt.Errorf("runtime root is not a directory: %s", absolute) }
 for _, candidate := range []string{"lib64/ld-linux-x86-64.so.2", "lib/ld-linux-x86-64.so.2", "lib/ld-linux-aarch64.so.1"} {
  loader := filepath.Join(absolute, candidate)
  if stat, err := os.Stat(loader); err == nil && !stat.IsDir() {
   return Runtime{Root:absolute, Loader:loader}, nil
  }
 }
 return Runtime{}, fmt.Errorf("%w: no supported glibc ELF loader under %s", ErrNotFound, absolute)
}
