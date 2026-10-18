# Veyra GLenv

Go-based foundation for glibc compatibility on musl-based Veyra Linux.

This is an early, non-invasive prototype. It does **not** install glibc, patch ELF binaries, modify the host loader, create mounts, or implement atomic updates.

## Build

```sh
go build -o glenv ./cmd/glenv
./glenv version
./glenv doctor --runtime /opt/glibc-env
go test ./...
```

`doctor` checks for a glibc ELF loader under a specified runtime root. A successful check only confirms that the loader path exists, **not** that applications will work.

## Design direction

- Keep glibc runtime files separate from mutable user state.
- Allow a future read-only, host-managed runtime to be shared into a user environment.
- Keep ordinary glibc application execution a goal; do not require a container per app.
- Leave deployment and OTA implementation to Veyra's future system manager.
- Support versioned runtimes without silently changing active application dependencies.

The existing [glenv](https://github.com/angyedz/glenv) project is the behavioral reference. Its existing system integration must be studied before implementing the launcher.

License file will be provided by the repository owner.
