# AGENTS.md

This file is for coding agents working in this repository.

## Repo Shape

- Backend Go module lives in `backend/`.
- Frontend lives in `frontend/`.
- Deployment assets live in `deploy/`.
- Production binary install on the current server uses:
  - host: `root@luokyme.com`
  - app dir: `/opt/sub2api`
  - systemd unit: `sub2api`
  - runtime config: `/opt/sub2api/config.yaml`

## Working Rules

- Do not assume the repo root is the Go module root. Run Go commands from `backend/`.
- Prefer targeted tests over full-suite runs unless the change is broad.
- Keep `prompt_cache_key` and real session identity separate.
- Be careful with production restarts: this service runs Redis cleanup on shutdown, so restarts can cold-start Redis-backed caches.

## Useful Commands

### Backend

```bash
cd backend
go test ./internal/service/... ./internal/handler/...
```

Build the backend binary:

```bash
cd backend
go build -ldflags='-s -w -X main.Version=$(tr -d '"'"'\r\n'"'"' < ./cmd/server/VERSION)' -trimpath -o /tmp/sub2api-deploy ./cmd/server
```

### Frontend

```bash
cd frontend
pnpm test
pnpm build
```

## How To Deploy

This section describes the current binary deployment flow used for `luokyme.com`.

### 1. Build locally

```bash
cd /home/lkm/ws/lkm/sub2api/backend
go build -ldflags='-s -w -X main.Version=$(tr -d '"'"'\r\n'"'"' < ./cmd/server/VERSION)' -trimpath -o /tmp/sub2api-deploy ./cmd/server
sha256sum /tmp/sub2api-deploy
```

### 2. Upload the new binary

Pick a descriptive suffix:

```bash
scp /tmp/sub2api-deploy root@luokyme.com:/opt/sub2api/sub2api.new-<tag>
```

Example tags:

- `compat-cache-20260418T231553`
- `fix-foo-YYYYMMDDTHHMMSS`

### 3. Backup current binary and switch

```bash
ssh root@luokyme.com '
set -euo pipefail
cd /opt/sub2api
ts=$(date +%Y%m%dT%H%M%S)
cp -a sub2api sub2api.bak.$ts.pre-deploy
install -m 755 -o root -g root sub2api.new-<tag> sub2api
systemctl restart sub2api
'
```

### 4. Verify service health

```bash
ssh root@luokyme.com '
systemctl is-active sub2api
systemctl status sub2api --no-pager -l | sed -n "1,25p"
journalctl -u sub2api -n 40 --no-pager
curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/
'
```

Expected minimum checks:

- `systemctl is-active sub2api` returns `active`
- local HTTP check returns `200`

### 5. Roll back if needed

List recent backups:

```bash
ssh root@luokyme.com 'cd /opt/sub2api && ls -1t sub2api.bak.* | head'
```

Restore one backup:

```bash
ssh root@luokyme.com '
set -euo pipefail
cd /opt/sub2api
install -m 755 -o root -g root sub2api.bak.<timestamp>.pre-deploy sub2api
systemctl restart sub2api
'
```

## Operational Notes

- The current systemd unit is `/etc/systemd/system/sub2api.service`.
- There is also a drop-in at `/etc/systemd/system/sub2api.service.d/debug-gateway.conf`.
- The drop-in currently enables gateway body debug logging:

```ini
Environment=SUB2API_DEBUG_GATEWAY_BODY=/opt/sub2api/gateway_debug.log
```

- When investigating prompt-cache behavior in production, inspect:
  - `journalctl -u sub2api`
  - `/opt/sub2api/gateway_debug.log`
  - `usage_logs` in PostgreSQL

## Change Validation For Prompt Cache Work

If a change touches compat caching or session routing, validate all three:

1. `prompt_cache_key` stays stable across later turns with the same stable prefix.
2. real session headers still distinguish separate conversations and forks.
3. upstream usage shows non-zero cache reads where expected.
