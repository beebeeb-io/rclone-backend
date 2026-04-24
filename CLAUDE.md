# beebeeb-io/rclone-backend

Rclone backend for Beebeeb. Go module.

## Build
```sh
go build ./...
```

## Test
```sh
go test ./...
```

## Run test CLI
```sh
BB_TOKEN=<token> go run ./cmd --list /
BB_TOKEN=<token> go run ./cmd --upload ./test.txt /test.txt
BB_TOKEN=<token> go run ./cmd --download /test.txt ./downloaded.txt
BB_TOKEN=<token> go run ./cmd --mkdir /backup/2026-04
BB_TOKEN=<token> go run ./cmd --delete /test.txt
```

## Structure

| File | Purpose |
|------|---------|
| backend.go | Core Fs and Object types mirroring rclone interfaces |
| api.go | HTTP client for the Beebeeb API (ListFiles, UploadFile, DownloadFile, DeleteFile, CreateFolder) |
| config.go | Configuration from env vars (BB_API_URL, BB_TOKEN) or rclone config map |
| cmd/main.go | Standalone test CLI binary |

## Keep shared docs in sync

When updating API calls or endpoint usage, ensure consistency with:
- `../../.claude/skills/beebeeb-api.md` — API reference
- `../server/CLAUDE.md` — server endpoint docs
