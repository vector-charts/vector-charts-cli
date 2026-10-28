# Vector Charts CLI (`vc`)

Command-line tool for connecting to Vector Charts Cloud and OEM instances: profile management, interactive login, API tokens, chart uploads, custom cells, and job status.

## Install

### Convenience script

```bash
curl -fsSL https://raw.githubusercontent.com/zydro-marine/vector-charts-cli/main/scripts/install.sh | bash
```

### Debian / Ubuntu

Download the `.deb` from [Releases](https://github.com/zydro-marine/vector-charts-cli/releases) and install:

```bash
sudo dpkg -i vc_*.deb
```

### Nix

```bash
nix profile install github:zydro-marine/vector-charts-cli
# or
nix run github:zydro-marine/vector-charts-cli -- version
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
vc server add cloud --url https://cloud.vectorcharts.com
vc server add local-oem --url http://localhost:9909

vc server use cloud
vc server login          # opens browser device flow
# or: vc server login --token <session-or-oem-api-token>
# or: vc server login --username admin

vc server discover       # find instances on the LAN (UDP multicast)
vc server list           # connectivity + auth probe
vc server whoami
vc charts upload ./chart.zip
vc token list
vc jobs status
```

Config lives under `~/.vector-charts/`:

- `config.yml` — profiles (`url`; `type` / `api_url` filled after login)
- `credentials.yml` — session / API tokens (mode 0600)

Override with `VECTOR_CHARTS_HOME`.

## Command groups

| Group | Purpose |
|-------|---------|
| `vc server` | Profiles, login/logout, connectivity |
| `vc token` | Portal API token CRUD |
| `vc charts` | Upload / uploads / cells / S-63 permit |
| `vc jobs` | List jobs and status snapshot |

Global flags: `--profile`, `-o table|json`, `--url`.

## Development

```bash
go build -o vc ./cmd/vc
go test ./...
```

Interactive login requires portal device-auth endpoints (`POST /api/portal/v1/auth/device/*`) and the SPA `/device` page.
