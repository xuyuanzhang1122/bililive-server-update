# Main Server Backup Restore API

## Problem

The current iOS backup / restore UI expects backup restore endpoints on the active `bililive-go` server:

- `POST /api/backups`
- `GET /api/backups/{id}`
- `POST /api/backups/restore`
- `GET /api/backups/restore/status/{job_id}`

The public update server can safely store and return backup packages, but it cannot write a user's local `config.yml`, change ports or paths, or restart the active recorder service.

## Current Solution

`bililive-server-update` implements the storage-compatible iOS endpoints:

- `POST /api/backups`
- `GET /api/backups/{id}`

It validates the iOS backup package shape:

- `schemaVersion == 1`
- `server.rpc_bind` is present
- `server.out_put_path` is present
- `server.app_data_path` is present
- `server.live_rooms[].url` is HTTP or HTTPS

For restore execution endpoints, this service returns explicit responses explaining that restore must run on the active `bililive-go-UI` machine:

- `POST /api/backups/restore` returns `405`
- `GET /api/backups/restore/status/{job_id}` returns `404`

The checked `bililive-go-UI` branch `feature/web-main-service-integration` currently contains the main-server implementation in `src/servers/backup_handler.go`:

- `POST /api/backups`
- `GET /api/backups/{id}`
- `POST /api/backups/restore`
- `GET /api/backups/restore/status/{job_id}`

That branch accepts backup ID or inline package, validates the package, writes restored config fields and `live_rooms`, persists config, and restarts when the bind address changes.

## Current Follow-Up

Keep this issue text as the handoff note for the main server branch. The solution is no longer “not implemented”; it is implemented in the current integration branch and should be reviewed/tested there before merging.

## Verification

In `bililive-server-update`:

```bash
gofmt -w ./cmd ./internal
go test ./...
```
