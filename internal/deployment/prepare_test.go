package deployment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareCopyPreservesFullGlenvAndSource(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "original-glenv")
	libdir := filepath.Join(source, "sysroot", "usr", "lib", "x86_64-linux-gnu")
	if err := os.MkdirAll(libdir, 0755); err != nil {
		t.Fatal(err)
	}
	libc := filepath.Join(libdir, "libc.so.6")
	if err := os.WriteFile(libc, []byte("glibc Debian fixture"), 0644); err != nil {
		t.Fatal(err)
	}
	bindir := filepath.Join(source, "lib")
	if err := os.MkdirAll(bindir, 0755); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(bindir, "glenv-wayland-filter.so")
	if err := os.WriteFile(shim, []byte("original explicit-sync shim fixture"), 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(source, "sysroot", "libc.so.6")
	if err := os.Symlink("/usr/lib/x86_64-linux-gnu/libc.so.6", link); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(base, "glenv-release-v1")
	if err := PrepareCopy(source, staging, "v1"); err != nil {
		t.Fatal(err)
	}
	manifest, err := readTree(staging)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Format != 2 || manifest.Version != "v1" {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	shimInfo, err := os.Stat(filepath.Join(staging, "lib", "glenv-wayland-filter.so"))
	if err != nil || shimInfo.Mode().Perm() != 0755 {
		t.Fatalf("Wayland shim mode not preserved: %v, %v", err, shimInfo)
	}
	stagedLink, err := os.Readlink(filepath.Join(staging, "sysroot", "libc.so.6"))
	if err != nil || stagedLink != "/usr/lib/x86_64-linux-gnu/libc.so.6" {
		t.Fatalf("Debian symlink lost: %q %v", stagedLink, err)
	}
	data, err := os.ReadFile(libc)
	if err != nil || string(data) != "glibc Debian fixture" {
		t.Fatalf("original glenv mutated: %q %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(source, "glenv-manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("source unexpectedly received a manifest: %v", err)
	}
	if err := PrepareCopy(source, staging, "v1"); err == nil {
		t.Fatal("overwrote existing staged release")
	}
}

func TestPrepareCopyRejectsUnsafePaths(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "original")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "payload"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(base, "symlink")
	if err := os.Symlink(source, linked); err != nil {
		t.Fatal(err)
	}
	if err := PrepareCopy(linked, filepath.Join(base, "out"), "v1"); err == nil {
		t.Fatal("accepted symlinked source")
	}
	if err := PrepareCopy(source, filepath.Join(source, "out"), "v1"); err == nil {
		t.Fatal("accepted output inside source")
	}
	if err := PrepareCopy(source, filepath.Join(base, "out"), "../traversal"); err == nil {
		t.Fatal("accepted invalid version")
	}
}

func TestPreparedReleaseCanStageWithoutMovingOriginal(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "original")
	if err := os.MkdirAll(filepath.Join(source, "sysroot"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "sysroot", "hello"), []byte("hello"), 0755); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(base, "staged")
	if err := PrepareCopy(source, staging, "v1"); err != nil {
		t.Fatal(err)
	}
	store := Store{Root: filepath.Join(base, "store")}
	if err := store.Stage(staging, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := store.Verify("v1"); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(source, "sysroot", "hello")); err != nil || string(data) != "hello" {
		t.Fatalf("original source moved or changed: %q %v", data, err)
	}
}
