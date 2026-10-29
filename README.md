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
vc charts upload ./chart.zip
vc charts cells list
vc permits list
vc store list
vc store enable au
vc token list
vc jobs status
```

Config lives under `~/.vector-charts/`:

- `config.yml` — profiles (`url`; `type` / `api_url` filled after login)
- `credentials.yml` — session / API tokens (mode 0600)

Override with `VECTOR_CHARTS_HOME`.

If `--profile` is omitted: the sole profile is used when only one exists, else `current_profile`, else `default`. URL may be positional on `vc server add` (`vc server add [name] <url>`).

## Command groups

| Group | Purpose |
|-------|---------|
| `vc server` | Profiles, login/logout, connectivity |
| `vc token` | Portal API token CRUD |
| `vc charts` | Upload / uploads / cells / S-63 user permit |
| `vc permits` | List licenses and S-63 cell permits (`licenses` alias) |
| `vc store` | Chart store subscriptions: list / enable / disable (`subscriptions` alias) |
| `vc jobs` | List jobs and status snapshot |

Global flags: `--profile`, `--format` / `-f` `table|json|plaintext` (`-o` alias), `--url`.

- `table` — aligned columns with a header rule (default)
- `json` — indented JSON
- `plaintext` — tab-separated values

## Development

```bash
go build -o vc ./cmd/vc
go test ./...
```

Interactive login requires portal device-auth endpoints (`POST /api/portal/v1/auth/device/*`) and the SPA `/device` page.
