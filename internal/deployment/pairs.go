package deployment

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DeploymentPair is a permanent host-deployment to glenv-generation mapping.
// It is local selection metadata, not a signed or authenticated OTA record.
type DeploymentPair struct {
	Format         int    `json:"format"`
	HostDeployment string `json:"host_deployment"`
	GlenvVersion   string `json:"glenv_version"`
}

const pairsDirectory = "deployment-pairs"

func (s Store) pairsDir(create bool) (string, error) {
	if err := s.verifyStoreRoot(); err != nil {
		return "", err
	}
	directory := filepath.Join(s.Root, pairsDirectory)
	if create {
		if err := s.Setup(); err != nil {
			return "", err
		}
		if err := os.Mkdir(directory, 0750); err != nil && !os.IsExist(err) {
			return "", err
		}
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("deployment-pairs must be a real directory")
	}
	return directory, nil
}

func pairPath(directory, hostID string) (string, error) {
	if !releaseName.MatchString(hostID) || hostID == "." || hostID == ".." {
		return "", fmt.Errorf("invalid Veyra host deployment identifier %q", hostID)
	}
	return filepath.Join(directory, hostID+".json"), nil
}

func readPairFile(filename, hostID string) (DeploymentPair, error) {
	var pair DeploymentPair
	info, err := os.Lstat(filename)
	if err != nil {
		return pair, err
	}
	if !info.Mode().IsRegular() || info.Size() > 4096 {
		return pair, errors.New("invalid deployment-pair file")
	}
	file, err := os.Open(filename)
	if err != nil {
		return pair, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 4097))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pair); err != nil {
		return pair, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return pair, errors.New("unexpected trailing deployment-pair data")
	}
	if pair.Format != 1 || pair.HostDeployment != hostID {
		return pair, errors.New("deployment-pair host identity mismatch")
	}
	if !releaseName.MatchString(pair.GlenvVersion) {
		return pair, errors.New("invalid glenv generation in deployment pair")
	}
	return pair, nil
}

// ReadPair resolves a specific Veyra host deployment without looking at the
// globally selected glenv 'current' release. This enables A/B host rollback.
func (s Store) ReadPair(hostID string) (DeploymentPair, error) {
	var pair DeploymentPair
	if _, err := pairPath(".", hostID); err != nil {
		return pair, err
	}
	directory, err := s.pairsDir(false)
	if err != nil {
		return pair, err
	}
	filename, err := pairPath(directory, hostID)
	if err != nil {
		return pair, err
	}
	return readPairFile(filename, hostID)
}

// PairHost creates a once-only mapping. Another glenv generation for an
// existing host ID is rejected: host deployment IDs must be immutable.
// This is not a trust signature or a privileged Veyra bootloader operation.
// A trusted update manager must hold an exclusive store lock and guarantee
// the release cannot be modified while it is verified and used.
func (s Store) PairHost(hostID, glenvVersion string) error {
	if _, err := pairPath(".", hostID); err != nil {
		return err
	}
	if _, err := s.release(glenvVersion); err != nil {
		return err
	}
	if err := s.Verify(glenvVersion); err != nil {
		return err
	}
	directory, err := s.pairsDir(true)
	if err != nil {
		return err
	}
	destination, err := pairPath(directory, hostID)
	if err != nil {
		return err
	}
	pair := DeploymentPair{Format: 1, HostDeployment: hostID, GlenvVersion: glenvVersion}
	data, err := json.MarshalIndent(pair, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	suffix, err := randomSuffix()
	if err != nil {
		return err
	}
	temporary := filepath.Join(directory, ".glenv-pair-"+suffix)
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	// Unlike Rename this cannot replace an existing deployment mapping.
	// Link gives atomic "create only if missing" semantics on this filesystem.
	if err := os.Link(temporary, destination); err != nil {
		if os.IsExist(err) {
			existing, readErr := readPairFile(destination, hostID)
			if readErr == nil && existing == pair {
				return nil // safe and idempotent
			}
			if readErr != nil {
				return fmt.Errorf("existing deployment pair is invalid: %w", readErr)
			}
			return fmt.Errorf("host deployment %s is already paired with another glenv version", hostID)
		}
		return err
	}
	if err := syncDir(directory); err != nil {
		return err
	}
	// Check that the manifest stayed valid after creation. If a concurrent
	// writer changed this tree, fail closed without rewriting the pair.
	if err := s.Verify(glenvVersion); err != nil {
		return fmt.Errorf("paired glenv generation failed re-verification: %w", err)
	}
	return nil
}

// PinPairedHost selects a stable generation for the host deployment selected
// by Veyra's trusted boot logic, without mutating the current/pair mapping.
func (s Store) PinPairedHost(hostID, destination string) error {
	pair, err := s.ReadPair(hostID)
	if err != nil {
		return err
	}
	return s.pinVersion(pair.GlenvVersion, destination)
}

// MarshalPair is intended for CLI diagnostics, not an authentication format.
func MarshalPair(pair DeploymentPair) ([]byte, error) {
	data, err := json.MarshalIndent(pair, "", "  ")
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(data), nil
}
