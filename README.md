<p align="center">
  <a href="https://beebeeb.io"><img src="https://beebeeb.io/assets/beebeeb-icon.png" alt="beebeeb" width="72" height="72" /></a>
</p>
<h1 align="center">beebeeb rclone-backend</h1>
<p align="center">An rclone backend for beebeeb — sync, copy, and FUSE-mount your encrypted vault from any machine.</p>
<p align="center"><strong>We can't recover your data. Not even if we wanted to.</strong> That's the point.</p>
<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-555.svg" alt="License: AGPL-3.0" /></a> &nbsp;
  <img src="https://img.shields.io/badge/go-1.25-555.svg" alt="Go 1.25" /> &nbsp;
  <a href="SECURITY.md"><img src="https://img.shields.io/badge/security-policy-555.svg" alt="Security policy" /></a>
</p>
<p align="center"><a href="https://beebeeb.io">Website</a> &nbsp;·&nbsp; <a href="https://beebeeb.io/security">How it works</a> &nbsp;·&nbsp; <a href="SECURITY.md">Report a vulnerability</a></p>
<p align="center"><sub>End-to-end encrypted cloud storage, built in Europe. Operated by Initlabs B.V., Wijchen, Netherlands.</sub></p>

---

This is an [rclone](https://rclone.org) backend for beebeeb. It lets you drive your encrypted vault with standard rclone commands — `sync`, `copy`, `ls`, `mount`, and the rest — so any server, cron job, or workstation can read and write your vault without a bespoke client.

Files are encrypted on the device before they reach the API. The beebeeb servers store ciphertext and never see plaintext; this backend speaks the same HTTP API the official clients use.

## Quick start

### 1. Install rclone

See [rclone.org/install](https://rclone.org/install/).

### 2. Configure the remote

```bash
rclone config create beebeeb bb \
  token=<your-session-token> \
  api_url=https://api.beebeeb.io
```

Or set environment variables instead:

```bash
export BB_TOKEN=<your-session-token>
export BB_API_URL=https://api.beebeeb.io
```

### 3. Use it

```bash
# List files in your vault
rclone ls beebeeb:

# Copy a local folder to the vault
rclone copy ./documents/ beebeeb:documents/

# Sync a backup directory
rclone sync /srv/backups/pg/ beebeeb:archive/pg/2026-04/ --progress --transfers 16

# Mount the vault as a local folder
rclone mount beebeeb: ~/vault --vfs-cache-mode full
```

A nightly backup is just a cron line:

```bash
# /etc/cron.d/bb-backup
0 3 * * *  root  rclone sync /srv/backups beebeeb:archive --delete-after
```

## Configuration

| Option    | Env var       | Default                 | Description                        |
|-----------|---------------|-------------------------|------------------------------------|
| `api_url` | `BB_API_URL`  | `http://localhost:3001` | beebeeb API base URL               |
| `token`   | `BB_TOKEN`    | *(required)*            | Session token for authentication   |

## Registering with rclone

This backend is meant to be compiled into rclone as a backend plugin. Add it as an import in your rclone fork or plugin build:

```go
import _ "github.com/beebeeb-io/rclone-backend"
```

Then build rclone as usual. The backend registers itself under the name `bb`.

## Building from source

```bash
go build ./...
go test ./...
```

### Standalone test CLI

You can exercise the backend without a full rclone build:

```bash
BB_TOKEN=<token> go run ./cmd --list /
BB_TOKEN=<token> go run ./cmd --upload ./test.txt /test.txt
BB_TOKEN=<token> go run ./cmd --download /test.txt ./downloaded.txt
BB_TOKEN=<token> go run ./cmd --mkdir /backup/2026-04
BB_TOKEN=<token> go run ./cmd --delete /test.txt
```

## Security

Found a vulnerability? Email **security@beebeeb.io** — see [SECURITY.md](SECURITY.md).

## Part of beebeeb

End-to-end encrypted, zero-knowledge cloud storage — made in Europe.
[core](https://github.com/beebeeb-io/core) · [cli](https://github.com/beebeeb-io/cli) · [web](https://github.com/beebeeb-io/web) · [mobile](https://github.com/beebeeb-io/mobile) · [desktop](https://github.com/beebeeb-io/desktop) · [website](https://beebeeb.io)

## License

[AGPL-3.0-or-later](LICENSE) — © Initlabs B.V. (KvK 95157565), Wijchen, Netherlands.
