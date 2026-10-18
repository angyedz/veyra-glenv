# Veyra GLenv

Experimental Go-based integration layer for the original [GLenv](https://github.com/angyedz/glenv) glibc subsystem on musl-based Veyra Linux.

**Veyra remains musl.** The glibc Debian sysroot is a compatibility stack for glibc applications, including graphics, audio, screen sharing, TLS, and the original Wayland shim. None of these original GLenv components should be discarded.

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

`run` invokes a selected runtime's dynamic loader with explicit library paths. It requires a compatible existing sysroot and is **not** yet a substitute for GLenv's complete system integration. It does not transparently intercept `execve` or automatically configure Wayland shim, PipeWire modules, NSS, graphics driver integration or absolute sysroot paths.

## Atomic Veyra integration

See [docs/veyra-integration.md](docs/veyra-integration.md) for the proposed contract: a host-owned, versioned, read-only runtime exposed to both host and user environments, activated with Veyra's future atomic deployment. **OTA is not implemented here.**

This repository currently makes no privileged system changes, no mount changes, and no changes to the original GLenv repository.

License file will be provided by the repository owner.
