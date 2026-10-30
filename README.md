# Vector Charts CLI (`vc`)

Command-line tool for connecting to Vector Charts Cloud and OEM instances: profile management, interactive login, API tokens, chart uploads, custom cells, licenses/permits, store subscriptions, and job status.

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
go build -o vc ./cmd/vc
sudo install -m 755 vc /usr/local/bin/vc
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
vc charts uploads create ./chart.zip
vc charts cells list
vc permits list
vc store list
vc store enable au
vc tokens list
vc jobs status
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
| `vc tokens` | Portal API token CRUD (`token` alias) |
| `vc charts` | Uploads / cells / S-63 user permit (`chart` alias) |
| `vc permits` | List licenses and S-63 cell permits (`licenses` alias) |
| `vc store` | Chart store subscriptions: list / enable / disable (`subscriptions` alias) |
| `vc jobs` | List jobs and status snapshot |

Global flags: `--profile`, `--format` / `-f` `table|json` (`-o` alias), `--url`.

- `table` — aligned columns (default)
- `json` — indented JSON

## Development

```bash
go build -o vc ./cmd/vc
go test ./...
```

### Dev symlink

To run the repo binary as `vc` without copying into `/usr/local/bin` on every rebuild, symlink it onto your `PATH`:

```bash
go build -o vc ./cmd/vc
sudo ln -sf "$(pwd)/vc" /usr/local/bin/vc
```

Rebuild with `go build -o vc ./cmd/vc` and the same `vc` on your `PATH` picks up the new binary. Remove the symlink (or replace it with a release install) when you are done developing.

If `/usr/local/bin` is not writable, use a user-local directory on your `PATH` instead:

```bash
mkdir -p "$HOME/.local/bin"
ln -sf "$(pwd)/vc" "$HOME/.local/bin/vc"
```

Interactive login requires portal device-auth endpoints (`POST /api/portal/v1/auth/device/*`) and the SPA `/device` page.
