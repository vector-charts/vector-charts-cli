# Vector Charts CLI (`vc`)

[![Build Status](https://github.com/vector-charts/vector-charts-cli/workflows/ci/badge.svg)](https://github.com/vector-charts/vector-charts-cli/actions?query=workflow%3Aci)

`vc` is a tool to control & monitor instances of [Vector Charts](https://vectorcharts.com/), a nautical chart & geospatial data platform. The CLI interfaces with the Vector Charts control plane in order to speed up common tasks such as chart & upload file management and job monitoring. 

In addition to being a useful utility, this project serves as an example of how to interact with Vector Charts software programmatically.

- [Requirements](https://github.com/vector-charts/vector-charts-cli#requirements)
- [Installation](https://github.com/vector-charts/vector-charts-cli#installation)
- [Configuration](https://github.com/vector-charts/vector-charts-cli#configuration)
- [Command Reference](https://github.com/vector-charts/vector-charts-cli#command-reference)
- [Developer Guide](https://github.com/vector-charts/vector-charts-cli#developer-guide)

## Requirements

- A Vector Charts Cloud account, or a locally-hosted Vector Charts OEM instance
- Linux or macOS (`x86_64` and `aarch64` are supported)

## Installation

Install with the convenience script:

```bash
curl -fsSL https://raw.githubusercontent.com/vector-charts/vector-charts-cli/main/scripts/install.sh | bash
```

On Debian / Ubuntu, download the `.deb` from [Releases](https://github.com/vector-charts/vector-charts-cli/releases) and install:

```bash
sudo dpkg -i vc_*.deb
```

With Nix:

```bash
nix profile install github:vector-charts/vector-charts-cli
# or
nix run github:vector-charts/vector-charts-cli -- version
```

From source:

```bash
go build -o build/vc ./cmd/vc
sudo install -m 755 build/vc /usr/local/bin/vc
```

### Shell completion

Shell completions help suggest `vc` subcommands.

For bash (requires the `bash-completion` package):

```bash
# current session
source <(vc cli completion bash)

# every new session (Linux)
vc cli completion bash | sudo tee /etc/bash_completion.d/vc > /dev/null

# every new session (macOS / Homebrew)
vc cli completion bash > "$(brew --prefix)/etc/bash_completion.d/vc"
```

For zsh:

```bash
# current session
source <(vc cli completion zsh)

# every new session - write into a directory on your $fpath
vc cli completion zsh > "${fpath[1]}/_vc"
```

If completions do not load, ensure `compinit` runs in your `~/.zshrc`.

## Configuration

`vc` uses named **profiles** to manage multiple Vector Charts instances. Each profile stores a URL and credentials. Profile configuration is stored in `~/.vector-charts` (or the path set by the `VECTOR_CHARTS_HOME` environment variable).

Before using the CLI to manage an instance, add one or more profiles:

```bash
# vc server add [name] [url] [flags]
vc server add cloud https://cloud.vectorcharts.com
```

Once you add a profile, or if authentication expires, refresh your local token:

```bash
# Authenticate via an interactive browser login flow
vc server login
connecting to server 'cloud'...
Open the following URL in a browser and confirm the code:

  https://cloud.vectorcharts.com/device?machineId=...

User code: ....-....

Waiting for approval...
Logged in as ... on profile "cloud"
```

```bash
# Authenticate with a Personal Access Token
vc server login --token <token-string>
```

To list all profiles:

```bash
vc server list
┌────────────┬────────────────────────────────┬───────────────────────────┬──────┐
│ NAME       │ URL                            │ STATUS                    │ AUTH │
├────────────┼────────────────────────────────┼───────────────────────────┼──────┤
│ local-oem* │ http://localhost:9909          │ offline, last seen 6h ago │ none │
│ cloud      │ https://cloud.vectorcharts.com │ online                    │ ok   │
└────────────┴────────────────────────────────┴───────────────────────────┴──────┘
```

To select an active profile as the default:

```bash
# vc server use <name> [flags]
vc server use cloud
Using profile "cloud"
```

Most `vc` commands interact with the active profile. Pass `--profile` to set it for a single command, or use `vc server use` to persist a default.

## Command Reference

### Global flags

Available on every command:

| Flag | Description |
|------|-------------|
| `--profile <name>` | Profile to use. If omitted, use the currently-set active profile, the profile named 'default', or the only profile if only one exists.|
| `-o, --output <table\|json\|plaintext>` | Output format (default `table`). `plaintext` is the same columns as `table`, without color, as comma-separated rows. |
| `-h, --help` | Help for the command. |

---

### `vc version`

Print CLI version.

```bash
vc version
```

---

### `vc server`

Manage connection profiles and authentication.

#### `vc server add [name] [url]`

Add a connection profile. If `name` is omitted, the profile is stored as `default`.

| Flag | Description |
|------|-------------|
| `--url <url>` | URL (alternative to positional URL). |

```bash
vc server add https://cloud.vectorcharts.com
vc server add cloud https://cloud.vectorcharts.com
vc server add oem --url http://localhost:9909
```

#### `vc server remove <name>`

Remove a connection profile (and its cached credentials).

#### `vc server use <name>`

Set the current profile.

#### `vc server info [name]`

Show profile details and probe connectivity and auth. Defaults to the current profile when `name` is omitted.

#### `vc server list`

List all profiles and probe connectivity / auth for each.

#### `vc server login`

Authenticate against the current profile. Modes:

1. **Device browser flow** (default) - opens a verification URL and waits for approval.
2. **`--token`** - Personal Access Token.
3. **`--username`** - username/email + interactive password prompt.

| Flag | Description |
|------|-------------|
| `--token <token>` | Personal Access Token. |
| `--username <name>` | Username/email for password login. |

```bash
vc server login
vc server login --token <token>
vc server login --username admin
```

#### `vc server logout`

Clear cached CLI credentials for the current profile (and attempt remote logout when a session is present).

---

### `vc token`

Manage Personal Access Tokens. Requires auth.

#### `vc token list`

List Personal Access Tokens.

#### `vc token create`

Create a Personal Access Token.

| Flag | Description | Default |
|------|-------------|---------|
| `--name <name>` | Token name | `CLI Token` |

```bash
vc token create
vc token create --name "CI bot"
```

#### `vc token update <id>`

Update a Personal Access Token name.

| Flag | Description |
|------|-------------|
| `--name <name>` | New token name |

```bash
vc token update <id> --name "Renamed"
```

#### `vc token delete <id>`

Delete a Personal Access Token.

---

### `vc chart`

Manage custom chart cells. Requires auth.

#### `vc chart list`

List custom cells.

#### `vc chart enable <id>`

Enable a custom cell.

#### `vc chart disable <id>`

Disable a custom cell.

#### `vc chart status <id>`

Get processing status for a custom cell.

```bash
vc chart list
vc chart enable <id>
vc chart disable <id>
vc chart status <id>
```

---

### `vc upload`

Create, list, status, sync, ftp-sync, or delete chart uploads. Requires auth.

Accepted file types: `.000`, `.zip`, `.txt`, `.prm`, `.h5`, `.vcm`.

#### `vc upload create <file>`

Also accepted as `vc upload add <file>`.

Upload a chart file for ingestion.

```bash
vc upload create ./chart.zip
vc upload add ./chart.zip
```

#### `vc upload list`

List uploads.

#### `vc upload status <id>`

Show status info for an upload (filename, status, size, chart counts, timestamps, logs).

```bash
vc upload status <id>
```

#### `vc upload sync [path]`

Scan a local folder for chart files missing from uploads, and upload them.

| Flag | Description |
|------|-------------|
| `--delete` | Delete remote uploads whose filename is not present locally |
| `--dry-run` | Show what would change without uploading/deleting |

```bash
vc upload sync
vc upload sync ./charts
vc upload sync ./charts --delete
vc upload sync ./charts --dry-run
```

#### `vc upload ftp-sync`

Download chart files from an S-63 FTP provider, then sync them as uploads.

| Flag | Description |
|------|-------------|
| `--ftp <host/path>` | FTP target (e.g. `ftp.example.com/ENC`) |
| `--username <name>` | FTP username |
| `--password <pass>` | FTP password (prefer the interactive prompt) |
| `--dir <path>` | Local download directory (default: temp dir) |
| `--prefix <text>` | Only include files whose remote path contains this substring |
| `--dry-run` | Download new files but do not upload |
| `--delete` | Delete remote uploads not present on the FTP server |

```bash
vc upload ftp-sync
vc upload ftp-sync --ftp ftp.example.com/ENC --username charts --dry-run
vc upload ftp-sync --dir ./ftp-cache
```

#### `vc upload delete <id>`

Delete an upload.

---

### `vc permit`

S-63 permit helpers. Requires auth.

#### `vc permit show-s63-user-permit`

Fetch the account S-63 user permit.

```bash
vc permit show-s63-user-permit
```

---

### `vc license`

List account chart licenses and S-63 cell permits. Requires auth.

#### `vc license list`

List store licenses and S-63 cell permits.

```bash
vc license list
```

---

### `vc store`

Manage chart store subscriptions. Requires auth.

#### `vc store list`

List chart store products and subscription state.

#### `vc store enable <product-code>`

Also accepted as `vc store subscribe <product-code>`.

Enable (subscribe to) a chart store product.

```bash
vc store enable au
vc store subscribe au
```

#### `vc store disable <product-code>`

Also accepted as `unsubscribe` or `cancel`.

Disable (unsubscribe from) a chart store product.

```bash
vc store disable au
vc store cancel au
```

---

### `vc job`

Inspect background jobs. Requires auth.

#### `vc job list`

List jobs. By default hides `completed` and `scheduled` jobs.

| Flag | Description | Default |
|------|-------------|---------|
| `--active-only` | Only pending/running jobs | `false` |
| `--all` | Include completed and scheduled jobs | `false` |

```bash
vc job list
vc job list --active-only
vc job list --all
```

#### `vc job status`

Show active jobs with progress summary.

```bash
vc job status
```

---

### `vc cli`

CLI utility commands.

#### `vc cli check-update`

Check GitHub Releases for a newer `vc` version. If an update is available, print the convenience install command.

```bash
vc cli check-update
```

#### `vc cli completion`

Generate shell autocompletion scripts.

##### `vc cli completion bash`

Generate the autocompletion script for bash.

##### `vc cli completion zsh`

Generate the autocompletion script for zsh.

```bash
vc cli completion bash
vc cli completion zsh
```

## Developer Guide

See [DEVELOPERS.md](DEVELOPERS.md).