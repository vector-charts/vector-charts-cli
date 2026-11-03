# `vc` command reference

Complete list of commands available in the Vector Charts CLI.

## Global flags

Available on every command:

| Flag | Description |
|------|-------------|
| `--profile <name>` | Profile to use. If omitted: sole profile when only one exists, else `current_profile`, else `default`. |
| `-f, --format <table\|json>` | Output format (default `table`). |
| `-o, --output <table\|json>` | Alias for `--format`. |
| `--url <url>` | Override portal URL for this invocation. |
| `-h, --help` | Help for the command. |

---

## `vc version`

Print CLI version.

```bash
vc version
```

---

## `vc server`

Manage connection profiles and authentication.

### `vc server add [name] [url]`

Add a connection profile (portal URL only). Cloud vs OEM and the tile API URL are resolved automatically after login.

If `name` is omitted, the profile is stored as `default`.

| Flag | Description |
|------|-------------|
| `--url <url>` | Portal base URL (alternative to positional URL). |

```bash
vc server add https://cloud.vectorcharts.com
vc server add cloud https://cloud.vectorcharts.com
vc server add oem --url http://localhost:9909
```

### `vc server remove <name>`

Remove a connection profile (and its cached credentials).

### `vc server use <name>`

Set the current profile.

### `vc server info [name]`

Show profile details and probe connectivity (portal + API health, auth). Defaults to the current profile when `name` is omitted.

### `vc server list`

List all profiles and probe connectivity / auth for each.

### `vc server whoami`

Show the authenticated portal user. Requires auth.

### `vc server login`

Authenticate against the current profile. Modes:

1. **Device browser flow** (default) - opens a verification URL and waits for approval.
2. **`--token`** - Personal Access Token (or legacy session / OEM API token).
3. **`--username`** - username/email + interactive password prompt.

| Flag | Description |
|------|-------------|
| `--token <token>` | Personal Access Token (or legacy session / OEM API token). |
| `--username <name>` | Portal username/email for password login. |

```bash
vc server login
vc server login --token <session-or-oem-api-token>
vc server login --username admin
```

### `vc server logout`

Clear cached CLI credentials for the current profile (and attempt remote logout when a session is present).

### `vc server discover`

Discover Vector Charts instances on the local network via UDP multicast beacons (`$VC-DISCOVER-1.0`). OEM containers typically need host networking for multicast to reach the LAN.

| Flag | Description | Default |
|------|-------------|---------|
| `--timeout <sec>` | Seconds to listen for beacons | `3` |
| `--group <addr>` | Multicast group | `239.254.0.2` |
| `--port <port>` | UDP port | `9910` |

```bash
vc server discover
vc server discover --timeout 5
```

---

## `vc token`

Manage portal API tokens. Requires auth.

### `vc token list`

List API tokens.

### `vc token create`

Create an API token.

| Flag | Description | Default |
|------|-------------|---------|
| `--name <name>` | Token name | `CLI Token` |

```bash
vc token create
vc token create --name "CI bot"
```

### `vc token update <id>`

Update an API token name.

| Flag | Description |
|------|-------------|
| `--name <name>` | New token name |

```bash
vc token update <id> --name "Renamed"
```

### `vc token delete <id>`

Delete an API token.

---

## `vc chart`

Manage custom chart cells. Requires auth.

### `vc chart list`

List custom cells.

### `vc chart enable <id>`

Enable a custom cell.

### `vc chart disable <id>`

Disable a custom cell.

### `vc chart status <id>`

Get processing status for a custom cell.

```bash
vc chart list
vc chart enable <id>
vc chart disable <id>
vc chart status <id>
```

---

## `vc upload`

Create, list, status, sync, or delete chart uploads. Requires auth.

Accepted file types: `.000`, `.zip`, `.txt`, `.prm`, `.h5`, `.vcm`.

### `vc upload create <file>`

Also accepted as `vc upload add <file>`.

Upload a chart file for ingestion.

```bash
vc upload create ./chart.zip
vc upload add ./chart.zip
```

### `vc upload list`

List uploads.

### `vc upload status <id>`

Show status info for an upload (filename, status, size, chart counts, timestamps, logs).

```bash
vc upload status <id>
```

### `vc upload sync [path]`

Scan a local folder (default: current directory) recursively for portal-compatible chart files, compare against remote uploads by sanitized filename, and upload any that are missing remotely.

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

### `vc upload ftp-sync`

Download chart files from an S-63 FTP provider, then sync them to portal uploads. Prompts for FTP host/path, username, and password unless given via flags.

The path is relative to the FTP login directory (`CWD`). Existing portal uploads are listed first; FTP files whose sanitized filename already exists are skipped (not re-downloaded).

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

### `vc upload delete <id>`

Delete an upload.

---

## `vc permit`

S-63 permit helpers. Requires auth.

### `vc permit show-s63-user-permit`

Fetch the account S-63 user permit.

```bash
vc permit show-s63-user-permit
```

---

## `vc license`

List account chart licenses and S-63 cell permits. Requires auth.

### `vc license list`

List store licenses and S-63 cell permits.

```bash
vc license list
```

---

## `vc store`

Manage chart store subscriptions. Requires auth.

### `vc store list`

List chart store products and subscription state.

### `vc store enable <product-code>`

Also accepted as `vc store subscribe <product-code>`.

Enable (subscribe to) a chart store product.

```bash
vc store enable au
vc store subscribe au
```

### `vc store disable <product-code>`

Also accepted as `unsubscribe` or `cancel`.

Disable (unsubscribe from) a chart store product.

```bash
vc store disable au
vc store cancel au
```

---

## `vc job`

Inspect background jobs. Requires auth.

### `vc job list`

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

### `vc job status`

Show active jobs with progress summary.

```bash
vc job status
```

---

## `vc cli`

CLI utility commands.

### `vc cli completion`

Generate shell autocompletion scripts.

#### `vc cli completion bash`

Generate the autocompletion script for bash.

#### `vc cli completion zsh`

Generate the autocompletion script for zsh.

```bash
vc cli completion bash
vc cli completion zsh
```
