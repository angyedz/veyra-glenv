package deployment

import (
 "crypto/ed25519"
 "errors"
 "fmt"
)

// VerifySignedVersion verifies the release named by a store generation with
// a public key supplied by trusted Veyra host metadata, not by the release.
func (s Store) VerifySignedVersion(version string,key ed25519.PublicKey)error{
 if err:=s.verifyStoreRoot();err!=nil{return err}
 if err:=s.verifyVersionsDir();err!=nil{return err}
 release,err:=s.release(version);if err!=nil{return err}
 manifest,err:=VerifySigned(release,key);if err!=nil{return err}
 if manifest.Version!=version{return fmt.Errorf("signed glenv manifest version %q does not match %q",manifest.Version,version)}
 return nil
}

// PinPairedHostSigned is an opt-in fail-closed boot path. It must be
// called with a trusted host deployment ID and a pinned public key.
// Pair files themselves remain locally managed and must reside on a
// trusted immutable host filesystem; this does not verify host boot state.
func (s Store) PinPairedHostSigned(hostID,destination string,key ed25519.PublicKey)error{
 if len(key)!=ed25519.PublicKeySize{return errors.New("missing or invalid trusted signing key")}
 pair,err:=s.ReadPair(hostID);if err!=nil{return err}
 if err:=s.VerifySignedVersion(pair.GlenvVersion,key);err!=nil{return err}
 return s.pinVersion(pair.GlenvVersion,destination)
}
