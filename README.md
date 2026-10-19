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
