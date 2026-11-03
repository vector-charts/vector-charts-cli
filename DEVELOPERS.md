# Developer guide

## Build and test

```bash
go build -o build/vc ./cmd/vc
go test ./...
```

## Nix

Requires [Nix](https://nixos.org/download/) with flakes enabled (`experimental-features = nix-command flakes` in `nix.conf`).

Enter the flake dev shell (Go, gopls, gotools, goreleaser):

```bash
nix develop
```

Then build and test as usual:

```bash
go build -o build/vc ./cmd/vc
go test ./...
```

Or build / run the packaged binary without entering the shell:

```bash
nix build            # result -> ./result/bin/vc
nix run -- version   # run without installing
```

## Dev symlink

To run the repo binary as `vc` without copying into `/usr/local/bin` on every rebuild, symlink it onto your `PATH`:

```bash
go build -o build/vc ./cmd/vc
sudo ln -sf "$(pwd)/build/vc" /usr/local/bin/vc
```

Rebuild with `go build -o build/vc ./cmd/vc` and the same `vc` on your `PATH` picks up the new binary. Remove the symlink (or replace it with a release install) when you are done developing.

If `/usr/local/bin` is not writable, use a user-local directory on your `PATH` instead:

```bash
mkdir -p "$HOME/.local/bin"
ln -sf "$(pwd)/build/vc" "$HOME/.local/bin/vc"
```

Interactive login requires portal device-auth endpoints (`POST /api/portal/v1/auth/device/*`) and the SPA `/device` page.

## Releasing

From an up-to-date `main`:

```bash
git checkout -b release/1.2.3
git push -u origin HEAD
```

Pushing `release/1.2.3` creates and pushes tag `v1.2.3`. That tag run builds with GoReleaser and uploads Linux/macOS archives plus `.deb` packages to a GitHub Release.

You can also push a `v*` tag directly, or run the **release** workflow from the Actions UI with a version input (`x.y.z`).
