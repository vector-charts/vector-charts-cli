# Vector Charts CLI (`vc`)

The Vector Charts CLI is a tool to interface & manage instances of [Vector Charts](https://vectorcharts.com/), a nautical chart & geospatial data platform. The CLI interfaces with the Vector Charts control plane in order to speed up common tasks such as chart & upload file management and job monitoring.

In addition to being a useful utility, this project serves as an example of how to interact with Vector Charts software programmatically.

## Install

### Convenience script

```bash
curl -fsSL https://raw.githubusercontent.com/vector-charts/vector-charts-cli/main/scripts/install.sh | bash
```

### Debian / Ubuntu

Download the `.deb` from [Releases](https://github.com/vector-charts/vector-charts-cli/releases) and install:

```bash
sudo dpkg -i vc_*.deb
```

### Nix

```bash
nix profile install github:vector-charts/vector-charts-cli
# or
nix run github:vector-charts/vector-charts-cli -- version
```

### From source

```bash
go build -o build/vc ./cmd/vc
sudo install -m 755 build/vc /usr/local/bin/vc
```

## Quick start

```bash
# Profiles only need a portal URL. Cloud vs OEM and the tile API host
# are resolved automatically after login (from account.planInstance).
vc server add https://cloud.vectorcharts.com
vc server add local-oem http://localhost:9909

vc server login          # opens browser device flow
# or: vc server login --token <session-or-oem-api-token>
# or: vc server login --username admin

vc server discover       # find instances on the LAN (UDP multicast)
vc server list           # connectivity + auth probe
vc server whoami
vc upload create ./chart.zip
vc chart list
vc permit show-s63-user-permit
vc license list
vc store list
vc store enable au
vc token list
vc job status
```

Config lives under `~/.vector-charts/`:

- `config.yml` — profiles (`url`; `type` / `api_url` filled after login; `last_seen` updated on successful contact)
- `credentials.yml` — session / API tokens (mode 0600)

Override with `VECTOR_CHARTS_HOME`.

If `--profile` is omitted: the sole profile is used when only one exists, else `current_profile`, else `default`. URL may be positional on `vc server add` (`vc server add [name] <url>`).

## Command groups

| Group | Purpose |
|-------|---------|
| `vc server` | Profiles, login/logout, connectivity |
| `vc token` | Portal API token CRUD |
| `vc chart` | Custom cells: list / enable / disable / status |
| `vc upload` | Chart upload create / list / status / sync / ftp-sync / delete |
| `vc permit` | S-63 permit helpers (`show-s63-user-permit`) |
| `vc license` | List licenses and S-63 cell permits |
| `vc store` | Chart store subscriptions: list / enable / disable |
| `vc job` | List jobs and status snapshot |
| `vc cli` | CLI utilities (shell completion) |

Global flags: `--profile`, `--format` / `-f` `table|json` (`-o` alias), `--url`.

- `table` — aligned columns (default)
- `json` — indented JSON

## Shell completion

Generate scripts with `vc cli completion <shell>`.

### Bash

Requires the `bash-completion` package.

```bash
# current session
source <(vc cli completion bash)

# every new session (Linux)
vc cli completion bash | sudo tee /etc/bash_completion.d/vc > /dev/null

# every new session (macOS / Homebrew)
vc cli completion bash > "$(brew --prefix)/etc/bash_completion.d/vc"
```

### Zsh

```bash
# current session
source <(vc cli completion zsh)

# every new session — write into a directory on your $fpath
vc cli completion zsh > "${fpath[1]}/_vc"
```

If completions do not load, ensure `compinit` runs in your `~/.zshrc`.

### Fish

```fish
# current session
vc cli completion fish | source

# every new session
vc cli completion fish > ~/.config/fish/completions/vc.fish
```

### PowerShell

```powershell
# current session
vc cli completion powershell | Out-String | Invoke-Expression

# every new session — add the line above to your PowerShell profile
# ($PROFILE), or write the script to a file and dot-source it from there
```

Start a new shell after installing permanent completions.

## Development

```bash
go build -o build/vc ./cmd/vc
go test ./...
```

### Nix

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

### Dev symlink

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
