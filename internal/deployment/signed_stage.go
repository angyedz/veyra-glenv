package deployment

import (
 "crypto/ed25519"
 "errors"
 "fmt"
 "path/filepath"
)

// StageSigned checks a detached Ed25519 signature and the complete release
// before admitting a staged runtime to the version store. This is an opt-in
// path; legacy Stage remains available for offline development fixtures.
//
// This operation requires an exclusively owned, quiescent source tree and
// trusted version store. It is NOT a protection against a privileged writer
// racing verification, and does not implement filesystem immutability.
func (s Store) StageSigned(source,version string,key ed25519.PublicKey)error{
 if len(key)!=ed25519.PublicKeySize{return errors.New("missing or invalid release signing key")}
 if _,err:=s.release(version);err!=nil{return err}
 source,err:=filepath.Abs(source);if err!=nil{return err}
 manifest,err:=VerifySigned(source,key);if err!=nil{return err}
 if manifest.Version!=version{return fmt.Errorf("signed release version %q does not match requested %q",manifest.Version,version)}
 if err:=s.Stage(source,version);err!=nil{return err}
 // Fail closed if anything about the signed release changed while it was
 // being admitted. Do not remove a suspect staged tree automatically.
 return s.VerifySignedVersion(version,key)
}
