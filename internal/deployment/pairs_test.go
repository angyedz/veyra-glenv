package deployment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPerDeploymentPairsSurviveABRollback(t *testing.T) {
	base := t.TempDir()
	store := Store{Root: filepath.Join(base, "store")}
	for _, version := range []string{"g1", "g2"} {
		if err := store.Stage(fixture(t, base, version), version); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.PairHost("host-A", "g1"); err != nil {
		t.Fatal(err)
	}
	if err := store.PairHost("host-B", "g2"); err != nil {
		t.Fatal(err)
	}
	// Pairing the same version is harmless even after subsequent host updates.
	if err := store.PairHost("host-A", "g1"); err != nil {
		t.Fatalf("idempotent pair: %v", err)
	}
	if err := store.PairHost("host-A", "g2"); err == nil {
		t.Fatal("host-A was silently remapped")
	}
	if err := store.Activate("g2"); err != nil {
		t.Fatal(err)
	}
	pin := filepath.Join(base, "boot", "runtime")
	if err := store.PinPairedHost("host-A", pin); err != nil {
		t.Fatal(err)
	}
	targetA, err := os.Readlink(pin)
	if err != nil || filepath.Base(targetA) != "g1" {
		t.Fatalf("initial A pin: %q: %v", targetA, err)
	}
	if err := store.PinPairedHost("host-B", pin); err != nil {
		t.Fatal(err)
	}
	targetB, err := os.Readlink(pin)
	if err != nil || filepath.Base(targetB) != "g2" {
		t.Fatalf("B pin: %q: %v", targetB, err)
	}
	// A/B boot rollback picks its OWN paired glenv generation, even though
	// 'current' still points to g2.
	if err := store.PinPairedHost("host-A", pin); err != nil {
		t.Fatal(err)
	}
	targetA, err = os.Readlink(pin)
	if err != nil || filepath.Base(targetA) != "g1" {
		t.Fatalf("rollback A pin: %q: %v", targetA, err)
	}
	if _, err := store.ReadPair("host-unknown"); !os.IsNotExist(err) {
		t.Fatalf("unknown host did not fail closed: %v", err)
	}
	if err := store.PinPairedHost("host-unknown", pin); err == nil {
		t.Fatal("booted unknown host")
	}
	if got, _ := os.Readlink(pin); got != targetA {
		t.Fatal("failed host selection changed existing pin")
	}
}

func TestPerDeploymentPairRejectsForeignSymlinks(t *testing.T) {
	base := t.TempDir()
	store := Store{Root: filepath.Join(base, "store")}
	if err := store.Stage(fixture(t, base, "g1"), "g1"); err != nil {
		t.Fatal(err)
	}
	if err := store.PairHost("../escape", "g1"); err == nil {
		t.Fatal("accepted traversal host ID")
	}
	if err := store.PairHost("host-A", "missing"); err == nil {
		t.Fatal("accepted absent runtime")
	}
	other := filepath.Join(base, "external")
	if err := os.Mkdir(other, 0755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(store.Root, pairsDirectory)
	if _, err := store.pairsDir(true); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, dir); err != nil {
		t.Fatal(err)
	}
	if err := store.PairHost("host-A", "g1"); err == nil {
		t.Fatal("followed foreign deployment-pairs directory")
	}
	if err := store.PinPairedHost("host-A", filepath.Join(base, "runtime")); err == nil {
		t.Fatal("followed symlinked deployment-pairs directory")
	}
}

func TestPairRejectsTamperedTarget(t *testing.T) {
	base := t.TempDir()
	store := Store{Root: filepath.Join(base, "store")}
	if err := store.Stage(fixture(t, base, "g1"), "g1"); err != nil {
		t.Fatal(err)
	}
	if err := store.PairHost("host-A", "g1"); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(store.Root, pairsDirectory, "host-A.json")
	if err := os.Remove(destination); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "nonexistent"), destination); err != nil {
		t.Fatal(err)
	}
	if err := store.PairHost("host-A", "g1"); err == nil {
		t.Fatal("accepted replaced mapping")
	}
	if _, err := store.ReadPair("host-A"); err == nil {
		t.Fatal("accepted symlink mapping")
	}
}
