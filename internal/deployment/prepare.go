package deployment

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PrepareCopy copies a complete, OFFLINE glenv prefix to a new staging tree.
// It never renames, deletes, or writes inside the source directory. In
// particular, it keeps Debian absolute symlinks as symlinks and copies the
// original Wayland filter without rebuilding or replacing its implementation.
//
// The source must be a trusted, quiescent directory. This is not a snapshot
// of a live, concurrently writable tree; no manifest can guarantee that.
func PrepareCopy(source, destination, version string) (err error) {
	if !releaseName.MatchString(version) || version == "." || version == ".." {
		return fmt.Errorf("invalid glenv version %q", version)
	}
	src, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	dst, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("glenv source must be a real directory, not a symlink")
	}
	// Do not walk a target under the source: the walk would recursively read
	// its own copies. A source nested within the destination cannot exist
	// when the destination is new, and Mkdir below rejects existing targets.
	rel, err := filepath.Rel(src, dst)
	if err != nil {
		return err
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return errors.New("staging destination must not be inside glenv source")
	}
	// Inventory before reading anything into the target so a changed source
	// will be detected by the final content-and-link verification.
	manifest, err := Inventory(src, version)
	if err != nil {
		return err
	}
	if err := os.Mkdir(dst, 0700); err != nil {
		return err // never overwrite an existing directory or symlink
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(dst)
		}
	}()
	type directoryMode struct {
		path string
		mode os.FileMode
	}
	modes := []directoryMode{{path: dst, mode: info.Mode().Perm()}}
	err = filepath.WalkDir(src, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == src {
			return nil
		}
		relative, err := filepath.Rel(src, name)
		if err != nil {
			return err
		}
		if filepath.ToSlash(relative) == "glenv-manifest.json" {
			return nil // replace any old inventory, not its content
		}
		target := filepath.Join(dst, relative)
		entryInfo, err := os.Lstat(name)
		if err != nil {
			return err
		}
		switch {
		case entryInfo.IsDir():
			if err := os.Mkdir(target, 0700); err != nil {
				return err
			}
			modes = append(modes, directoryMode{path: target, mode: entryInfo.Mode().Perm()})
			return nil
		case entryInfo.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(name)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case entryInfo.Mode().IsRegular():
			return copyRegular(name, target, entryInfo.Mode().Perm())
		default:
			return fmt.Errorf("unsupported glenv source entry: %s", name)
		}
	})
	if err != nil {
		return err
	}
	if err := VerifyTree(dst, manifest); err != nil {
		return fmt.Errorf("copied glenv tree does not match source inventory: %w", err)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dst, "glenv-manifest.json"), data, 0644); err != nil {
		return err
	}
	// Do not change source permissions; restore each copied directory from
	// deepest to shallowest only after all writes have finished.
	for i := len(modes) - 1; i >= 0; i-- {
		if err := os.Chmod(modes[i].path, modes[i].mode); err != nil {
			return err
		}
	}
	complete = true
	return nil
}

func copyRegular(source, target string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	if err := output.Chmod(mode); err != nil {
		output.Close()
		return err
	}
	if err := output.Sync(); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}
