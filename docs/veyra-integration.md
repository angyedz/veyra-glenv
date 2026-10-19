# Veyra glenv integration contract (design, not installed)

## Non-negotiable properties

- Veyra host and user environment remain musl-based.
- glenv provides a **glibc compatibility userspace**, not a replacement libc for Veyra.
- Preserve the original glenv Debian sysroot and its functional components: GTK, Mesa/EGL/GLVND, PipeWire client libraries **and dlopen modules**, TLS/NSS, and the existing Wayland explicit-sync shim.
- Never modify the original `angyedz/glenv` repository as part of this port.
- No host loader replacement, bind mounts, fstab changes, privileged service installation, or OTA activation from the current CLI.

## Proposed filesystem contract

The host owns versioned immutable trees. The example paths are not yet implemented:

```
/usr/lib/veyra/glenv/versions/<id>/sysroot/  # host-managed immutable runtime
/run/glenv/runtime/                         # selected runtime visible in both environments
```

Host OpenRC and user OpenRC must see the **same selected runtime** at the same path. The host provisions a read-only mount into the user environment before its init starts. The user namespace must not gain permission to mutate host mounts or the runtime store.

The ordinary Linux ELF `PT_INTERP` path is **not relocatable**: a normal `execve` requires the requested interpreter to exist at that absolute path in the process's mount namespace. A stable loader entry point in *both* environments, or an explicit launcher, is necessary. The current `glenv run` is only a non-privileged prototype; it does **not** provide transparent direct exec.

## Updates and rollback

Prepare and validate a complete immutable runtime tree offline. Include the loader, sysroot libraries, graphics stack, PipeWire modules, Wayland shim and relevant metadata in the same version. Activate the new host deployment and its glenv version **together at boot**. On rollback, activate the previous matching version. Keep old trees while any running process or rollback target depends on them. Never mutate a live runtime tree in place.

Host OTA must not change the user environment's APK database or installed applications. Runtime compatibility needs ABI tests against real Wayland/KWin, PipeWire, screen sharing, and graphics hardware before activation.

## Important loader limitations

`ld-linux --library-path` only supplies library search paths. It does not automatically solve absolute paths, `dlopen` modules, NSS, locales, certificates, GLVND vendor JSON, GPU driver consistency, `ld.so.cache`, or Wayland shim injection. These require an explicit integration design and tests. **Do not ship this launcher as a replacement for original glenv yet.**

The legacy `/opt/glibc-env/sysroot` is supported as an input for experiments; this repository does not install or update it.

## Staged runtime manifest (prototype)

The `internal/deployment` package verifies `glenv-manifest.json` with `format: 1`, a version, and an explicit list of regular files with SHA-256 digests. Verification is read-only and must pass before a later activation component is permitted to select the version. It is **not** a complete directory integrity guarantee: unlisted files and symlinks are not validated. Debian sysroots require a future manifest format covering symlink targets and the entire release tree before any production activation.

No privileged deployment or OTA switch is implemented.

## Complete tree verification (format 3)

The new `deployment.Inventory(root, version)` produces a deterministic inventory of regular files (SHA-256), directories and symbolic links (literal link targets). `deployment.VerifyTree` detects changed content, changed link targets, and unexpected/missing tree entries without following symlinks. The `glenv-manifest.json` file is excluded from the inventory to avoid recursive hashing. This is a **local integrity check**, not proof of publisher authenticity; an OTA implementation must authenticate its signed release metadata and verify the release tree before activation.

The current implementation **does not install, mount, switch, or garbage-collect runtimes**. Immutable version storage, host/user mount namespace provisioning, boot selection and rollback are still separate work items. In particular, an inventory is not a secure immutable snapshot while another process can mutate the source concurrently: stage under trusted ownership and prevent writes before verification and activation.

## Version-store prototype (implemented)

The `glenv release` CLI now offers `inventory`, `stage`, `verify`, `activate`, `pin`, `rollback` and `status`. The store is configurable with `--store` (default `/var/lib/veyra/glenv`). A staged source directory must contain `glenv-manifest.json` format 3; generate that file with `glenv release inventory <dir> <version> > <dir>/glenv-manifest.json`. `stage` verifies the complete tree and renames it into `versions/<version>` on the **same filesystem**. `activate` atomically replaces the `current` symlink; `previous` keeps the last selected version for an operator-initiated rollback. `pin` creates a boot-specific symlink to a verified generation and does **not** follow subsequent selection updates.

This is **not yet an atomic Veyra host OTA implementation**. Two selector symlinks are not one crash-consistent multi-object transaction; a boot coordinator must pair host deployment and runtime release in one validated boot record. The pin itself is not a bind mount. No mount namespace setup, read-only enforcement, signature verification, unattended update, or original glenv Debian package builder is installed here. Do not run `activate` on a production machine before integrating a privileged boot coordinator, signature validation, and protected read-only generation storage.

For a two-environment boot, the host boot coordinator should resolve the validated release once, then expose that **same pinned generation** read-only into both host and user namespaces at a stable path. Existing processes must stay on their old generation until exit/reboot. Store ownership and permissions must prevent users from editing manifests, release trees or selection links.

## Persistent A/B deployment-to-glenv mapping (development)

The original `bind-host` command maintains a **single mutable boot-binding.json** for one selected host deployment. That file is insufficient for a direct host rollback because it forgets previous host IDs.

Prefer the new **per-deployment**, create-only mapping for A/B development:

```sh
glenv release --store /var/lib/veyra/glenv pair-host veyra-A glenv-1
glenv release --store /var/lib/veyra/glenv pair-host veyra-B glenv-2
glenv release --store /var/lib/veyra/glenv pair-status veyra-A
# Trusted boot coordinator supplies selected host slot ID:
glenv release --store /var/lib/veyra/glenv boot-pair-pin veyra-A /run/glenv/runtime
```

`pair-host` writes `deployment-pairs/<hostID>.json` using an atomic **create-if-absent** hard-link transaction after validating that the referenced glenv tree exists and passes its manifest. Repeating exactly the same mapping is allowed, but repointing an existing Veyra host ID to another runtime is rejected. When Veyra's bootloader rolls back from slot B to slot A, `boot-pair-pin veyra-A` selects A's **original** runtime even if glenv's global `current` pointer still points to B.

`boot-pair-pin` is a selector, **not a signed boot manager**. Veyra must provide the authentic selected host deployment ID, read-only versions, signature verification, durable image-update orchestration and read-only mount propagation before this can become a production OTA system. For legacy tests, `bind-host` and `boot-pin` remain available, but Veyra integration should use the per-host commands.

## Offline preparation and mode-aware integrity

The safe preparation workflow is now:

```sh
# The source is an OFFLINE, quiescent original glenv prefix. It is NOT moved.
glenv release prepare-copy /offline/glenv-prefix /staging/glenv-v3 glenv-v3
# A complete mode-aware manifest has been written to the new copy:
glenv release --store /var/lib/veyra/glenv stage /staging/glenv-v3 glenv-v3
glenv release --store /var/lib/veyra/glenv verify glenv-v3
```

`prepare-copy` preserves Debian absolute symlink text, regular files, executable permissions, directory permissions and the original `lib/glenv-wayland-filter.so`, without modifying the source. Its **format-3** inventory includes SHA-256 hashes, link targets and Unix permission bits. Changes to file modes (including execute/setuid) fail verification. Older **format-2** manifests are still supported for compatibility, but they cannot detect permission changes and should not be accepted for a new production OTA release.

This operation only prepares a local copy. It does not fetch Debian packages, resolve dependencies, authenticate upstream glenv, create a safe live snapshot, or activate an update. The source must not be written while it is inventoried and copied; a read-only offline source is strongly preferred. Do not run the original glenv installer into a live Veyra root as part of this command.

## Detached release signature verification (development)

A new opt-in verifier accepts a format-3 `glenv-manifest.json`, the detached `glenv-manifest.json.sig` (standard Base64-encoded 64-byte Ed25519 signature of the **exact manifest bytes**), and a **separately trusted** Ed25519 public key file (standard Base64-encoded 32-byte public key):

```sh
glenv release verify-signed /staging/glenv-v1 /path/to/trusted-glenv-ed25519.pub
```

The verifier checks the signature **before** interpreting the manifest, then hashes all listed files and verifies the complete tree, directory and file modes, and symlink targets. Signed metadata files are excluded from the tree inventory to prevent self-referential hashes. Missing, malformed, or untrusted signatures fail closed.

The opt-in `stage-signed` command authenticates the release before moving it into the version store and checks the signature again after staging. `boot-pair-pin-signed` rechecks the signer before selecting the host's runtime. **Legacy `stage`, `pair-host`, and `boot-pair-pin` can still accept unsigned releases**; the local host pair is not itself signed. Veyra's eventual privileged updater must pin a trusted public key through a signed host deployment, reject unsigned stages, protect immutable runtime storage, and support key rotation and replay-resistant deployment IDs. Never obtain the trust key from within the untrusted glenv release itself.

### Authenticated boot pin (opt-in)

A future Veyra boot coordinator that already has a **trusted, pinned Ed25519 public key** can select a paired runtime with an additional signature verification gate:

```sh
glenv release --store /var/lib/veyra/glenv boot-pair-pin-signed veyra-host-v1 /run/glenv/selected /etc/veyra/trust/glenv-ed25519.pub
```

Unlike `boot-pair-pin`, this refuses absent signatures and format-2 releases, rejects unknown public keys, and checks the selected release's complete format-3 manifest before pinning. It never generates or downloads a key. This is not switched on automatically: the ISO does not yet provision trusted host deployment metadata or a trusted glenv signing key. The boot coordinator must hold the version store immutable while verifying and mounting, must authenticate the host ID and deployment mapping itself, and must mount the exact pinned generation read-only into host and user environments.

### Verified signed-runtime mount tests

Veyra's optional early-boot scripts accept `VEYRA_GLENV_REQUIRE_SIGNATURE=1` and a **separately provisioned** `VEYRA_GLENV_TRUST_KEY`. In strict mode both boot selection and pre-mount checks verify the signature. [Veyra CI](https://github.com/angyedz/Veyra-Linux/actions/runs/39212313713) exercised a real signed generation mounted read-only into both host/user views inside an isolated mount namespace, then reverified the Ed25519 signature through each mount.

These tests do not establish a production trust chain. The selected Veyra host ID and deployment mapping must originate in authenticated boot metadata; the public key and release storage must be protected from modification, and the host/user mount namespace configuration must be coordinated by a privileged updater. **Read-only bind mounts alone do not make the underlying source immutable** or eliminate races with a privileged updater.
