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

## Complete tree verification (format 2)

The new `deployment.Inventory(root, version)` produces a deterministic inventory of regular files (SHA-256), directories and symbolic links (literal link targets). `deployment.VerifyTree` detects changed content, changed link targets, and unexpected/missing tree entries without following symlinks. The `glenv-manifest.json` file is excluded from the inventory to avoid recursive hashing. This is a **local integrity check**, not proof of publisher authenticity; an OTA implementation must authenticate its signed release metadata and verify the release tree before activation.

The current implementation **does not install, mount, switch, or garbage-collect runtimes**. Immutable version storage, host/user mount namespace provisioning, boot selection and rollback are still separate work items. In particular, an inventory is not a secure immutable snapshot while another process can mutate the source concurrently: stage under trusted ownership and prevent writes before verification and activation.

## Version-store prototype (implemented)

The `glenv release` CLI now offers `inventory`, `stage`, `verify`, `activate`, `pin`, `rollback` and `status`. The store is configurable with `--store` (default `/var/lib/veyra/glenv`). A staged source directory must contain `glenv-manifest.json` format 2; generate that file with `glenv release inventory <dir> <version> > <dir>/glenv-manifest.json`. `stage` verifies the complete tree and renames it into `versions/<version>` on the **same filesystem**. `activate` atomically replaces the `current` symlink; `previous` keeps the last selected version for an operator-initiated rollback. `pin` creates a boot-specific symlink to a verified generation and does **not** follow subsequent selection updates.

This is **not yet an atomic Veyra host OTA implementation**. Two selector symlinks are not one crash-consistent multi-object transaction; a boot coordinator must pair host deployment and runtime release in one validated boot record. The pin itself is not a bind mount. No mount namespace setup, read-only enforcement, signature verification, unattended update, or original glenv Debian package builder is installed here. Do not run `activate` on a production machine before integrating a privileged boot coordinator, signature validation, and protected read-only generation storage.

For a two-environment boot, the host boot coordinator should resolve the validated release once, then expose that **same pinned generation** read-only into both host and user namespaces at a stable path. Existing processes must stay on their old generation until exit/reboot. Store ownership and permissions must prevent users from editing manifests, release trees or selection links.
