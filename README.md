# Veyra glenv

Experimental Go-based integration layer for the original [glenv](https://github.com/angyedz/glenv) glibc subsystem on musl-based Veyra Linux.

**Veyra remains musl.** The glibc Debian sysroot is a compatibility stack for glibc applications, including graphics, audio, screen sharing, TLS, and the original Wayland shim. None of these original glenv components should be discarded.

## Current commands

```sh
go test ./...
go build -o glenv ./cmd/glenv
./glenv version
./glenv inspect /path/to/glibc-app
./glenv doctor --runtime /opt/glibc-env
./glenv run --runtime /opt/glibc-env --dry-run /path/to/glibc-app
./glenv run --runtime /opt/glibc-env /path/to/glibc-app
```

`run` invokes a selected runtime's dynamic loader with explicit library paths. It requires a compatible existing sysroot and is **not** yet a substitute for glenv's complete system integration. It does not transparently intercept `execve` or automatically configure Wayland shim, PipeWire modules, NSS, graphics driver integration or absolute sysroot paths.

## Atomic Veyra integration

See [docs/veyra-integration.md](docs/veyra-integration.md) for the proposed contract: a host-owned, versioned, read-only runtime exposed to both host and user environments, activated with Veyra's future atomic deployment. **OTA is not implemented here.**

This repository currently makes no privileged system changes, no mount changes, and no changes to the original glenv repository.

License file will be provided by the repository owner.

## Versioned runtime operations (development only)

The `release` commands manage verified staged generations. `inventory` emits a complete format-2 manifest, including regular-file SHA-256 hashes, directories, and Debian symlink targets. The release store can stage and verify a tree, select `current`, retain `previous`, roll back selection, and pin the verified generation for a boot session.

```sh
# Work only in a disposable test directory until a privileged Veyra boot
# coordinator and signed OTA metadata are implemented.
glenv release inventory /tmp/glenv-v1 v1 > /tmp/glenv-v1/glenv-manifest.json
glenv release --store /tmp/glenv-store stage /tmp/glenv-v1 v1
glenv release --store /tmp/glenv-store verify v1
glenv release --store /tmp/glenv-store activate v1
glenv release --store /tmp/glenv-store pin /tmp/glenv-boot/runtime
glenv release --store /tmp/glenv-store status
```

See [integration design](docs/veyra-integration.md) for important security and atomicity limitations. **Do not install this prototype over the original glenv runtime**: it has no verified signature chain, managed immutable mounts, system-wide boot-deployment transaction, or completed Wayland/PipeWire functional test suite.

## Boot pairing with a Veyra deployment (prototype)

One atomically replaced `boot-binding.json` stores both the Veyra host deployment ID and the required glenv release version. The boot coordinator pins that precise glenv version only when its host ID matches:

```sh
glenv release --store /var/lib/veyra/glenv bind-host veyra-host-v1 glenv-v1
glenv release --store /var/lib/veyra/glenv boot-status
glenv release --store /var/lib/veyra/glenv boot-pin veyra-host-v1 /run/glenv/runtime
```

This pair of IDs is not a signed Veyra OTA or bootloader transaction. Veyra must independently authenticate the host generation, keep the glenv runtime immutable, expose the same boot-pinned generation read-only to both environments, and validate the full glibc/Wayland/PipeWire stack before any production rollout.

GitHub Actions builds a **development-only**, statically linked Alpine/musl `glenv` binary and uploads a tarball with SHA-256 checksum. Retrieve it from the matching [CI run](https://github.com/angyedz/veyra-glenv/actions/workflows/go.yml); it is not a signed production package and does not replace the existing glenv installation.

## A/B host rollback bindings

For the next-generation Veyra boot coordinator, use the **immutable per-deployment** commands instead of the legacy mutable `bind-host` singleton:

```sh
glenv release --store /var/lib/veyra/glenv pair-host veyra-A glenv-v1
glenv release --store /var/lib/veyra/glenv pair-host veyra-B glenv-v2
glenv release --store /var/lib/veyra/glenv pair-status veyra-A
glenv release --store /var/lib/veyra/glenv boot-pair-pin veyra-A /run/glenv/runtime
```

Each host ID has its own create-only mapping; it cannot silently be retargeted to another glenv generation. Selecting an older host ID after a rollback pins its matching previous runtime, regardless of the `current` selector. These records are not signed, and boot pinning remains a **prototype** until Veyra implements authenticated boot metadata, trusted immutable storage and read-only runtime mounts for both environments.
