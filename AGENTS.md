# AGENTS.md

This file is for future agents working inside `bililive-server-update`.

## Project Boundary

Only modify files inside this repository:

```text
/Users/xu/Documents/GitHub/bililive-server-update
```

Do not edit sibling repositories such as `bililive-go-UI` or `bililive-ios` from this project. If a required change belongs to those projects, document the required API/client change in `docs/` here.

## Purpose

This service is the external update / mirror / backup server for the bililive ecosystem.

It does not record livestreams and does not replace `bililive-go-UI`.

Responsibilities:

- Serve install manifests for unified curl-based installation.
- Store release/tool catalog metadata for GitHub mirror, binary, Docker, npm, ffmpeg, headless browser, and future tools.
- Provide a common doctor response shape for installers.
- Store iOS/Web uploaded configuration backups.
- Return restore plans for configuration recovery.

Primary docs:

- `docs/server-plan.md`: server-side planning and ownership split.
- `docs/api-contract.md`: all APIs iOS/Web/backend need to know about.
- `docs/install-workflow.md`: unified installer flow.
- `docs/ios-web-integration.md`: client integration notes.

## Run Locally

```bash
go run ./cmd/server
```

Default:

- Address: `:8090`
- Data directory: `./data`
- Catalog file: `./data/catalog.json`
- Backups: `./data/backups/*.json`

Useful environment variables:

```bash
BLSU_ADDR=:8090
BLSU_DATA_DIR=./data
BLSU_ADMIN_TOKEN=change-me
BLSU_PUBLIC_BASE_URL=https://update.example.com
BLSU_GITHUB_REPO=xuyuanzhang1122/bililive-go-UI
```

`BLSU_ADMIN_TOKEN` is required for catalog mutation APIs.

## Verify

Before handing work back:

```bash
gofmt -w ./cmd ./internal
go test ./...
```

If you add shell scripts, also run:

```bash
sh -n path/to/script.sh
```

## API Summary

Public:

- `GET /health`
- `GET /install.sh`
- `GET /install.ps1`
- `GET /api/v1/install/manifest`
- `POST /api/v1/doctor`
- `GET /api/v1/catalog`
- `POST /api/v1/backups`
- `GET /api/v1/backups/{id}`
- `POST /api/v1/backups/{id}/restore-request`

Admin, requiring `Authorization: Bearer <BLSU_ADMIN_TOKEN>`:

- `POST /api/v1/catalog/sync-github`
- `PUT /api/v1/catalog/releases`
- `PUT /api/v1/catalog/tools`

When adding APIs, update `docs/api-contract.md` in the same change.

## Nginx Reverse Proxy

Use the example at `deploy/nginx.example.conf`.

Minimum reverse proxy shape:

```nginx
server {
    listen 443 ssl http2;
    server_name update.example.com;

    ssl_certificate /etc/letsencrypt/live/update.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/update.example.com/privkey.pem;

    client_max_body_size 50m;

    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Recommended production notes:

- Set `BLSU_PUBLIC_BASE_URL=https://update.example.com` so generated manifest URLs are stable.
- Limit admin endpoints by IP or require VPN if possible.
- Keep `client_max_body_size` large enough for iOS backup packages.
- Terminate TLS at nginx; keep the Go service bound to `127.0.0.1:8090` when deployed on a single machine.

## Systemd Example

Use `deploy/bililive-server-update.service` as a starting point.

Production values that must be changed:

- `WorkingDirectory`
- `ExecStart`
- `BLSU_ADMIN_TOKEN`
- `BLSU_PUBLIC_BASE_URL`
- `BLSU_DATA_DIR`

## Project-Level Problems and Issues

If you find a project-level problem:

1. Fix it in this repository when it is safe and in scope.
2. Run verification (`gofmt`, `go test ./...`, plus any relevant script checks).
3. Only after the fix is verified, create or draft an issue.
4. Do not mark it as unfinished. The issue should state:
   - Problem
   - Current implemented solution
   - Verification performed
   - Remaining follow-up, if any

If GitHub remote or credentials are unavailable, add the issue text to `docs/issues/` and mention that it is ready to submit.

## Coding Style

- Keep the service small and boring: standard library first.
- Persist operator-editable state as JSON under `data/`.
- Avoid adding databases unless the JSON store becomes a clear bottleneck.
- Keep install and restore APIs explicit; do not let this public update server directly mutate a user's bililive machine.
- Real machine mutations belong in `bililive-go-UI` local/admin APIs and should be documented here instead of implemented here.

