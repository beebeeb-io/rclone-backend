<p align="center">
  <h3 align="center">Beebeeb Rclone Backend</h3>
  <p align="center">An rclone backend for Beebeeb, letting you sync, copy, and FUSE-mount your encrypted vault.</p>
</p>

![License](https://img.shields.io/badge/license-AGPL--3.0-blue)
![Go](https://img.shields.io/badge/go-%3E%3D1.22-00ADD8)
![CI](https://img.shields.io/github/actions/workflow/status/beebeeb-io/rclone-backend/ci.yml)

## What is Beebeeb?

Beebeeb is an end-to-end encrypted file vault. Your files are encrypted client-side before they leave your device, and only you hold the keys. Beebeeb servers store ciphertext and never see plaintext.

This module implements an **rclone backend** so you can interact with your vault using standard rclone commands: `sync`, `copy`, `ls`, `mount`, and more.

## Quick start

### 1. Install rclone

See [rclone.org/install](https://rclone.org/install/).

### 2. Configure the Beebeeb remote

```bash
rclone config create beebeeb bb \
  token=<your-session-token> \
  api_url=https://api.beebeeb.io
```

Or set environment variables:
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

# Sync a remote backup
rclone sync /srv/backups/pg/ beebeeb:archive/pg/2026-04/ --progress --transfers 16

# Mount the vault as a local folder
rclone mount beebeeb: ~/vault --vfs-cache-mode full
```

## Usage examples

### Daily backup cron

```bash
# /etc/cron.d/bb-backup
0 3 * * *  root  rclone sync /srv/backups beebeeb:archive --delete-after
```

### Sync with progress

```bash
rclone sync /srv/backups/pg/ beebeeb:archive/pg/2026-04/ \
  --progress --transfers 16
```

### Mount as FUSE filesystem

```bash
rclone mount beebeeb: ~/vault --vfs-cache-mode full --daemon
```

## Standalone test CLI

You can exercise the backend without a full rclone build:

```bash
BB_TOKEN=<token> go run ./cmd --list /
BB_TOKEN=<token> go run ./cmd --upload ./test.txt /test.txt
BB_TOKEN=<token> go run ./cmd --download /test.txt ./downloaded.txt
BB_TOKEN=<token> go run ./cmd --mkdir /backup/2026-04
BB_TOKEN=<token> go run ./cmd --delete /test.txt
```

## Building from source

```bash
go build ./...
go test ./...
```

## Registering with rclone

This backend is designed to be compiled into rclone as a backend plugin. Add it as an import in your rclone fork:

```go
import _ "github.com/beebeeb-io/rclone-backend"
```

Then build rclone as usual. The backend registers itself under the name `bb`.

## Configuration options

| Option | Env var | Default | Description |
|--------|---------|---------|-------------|
| `api_url` | `BB_API_URL` | `http://localhost:3001` | Beebeeb API base URL |
| `token` | `BB_TOKEN` | *(required)* | Session token for authentication |

## License

[AGPL-3.0](LICENSE) -- Copyright 2026 Beebeeb
