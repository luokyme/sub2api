# AGENTS.md

本文件面向在本仓库中工作的编码代理。

## 仓库结构

- 后端 Go 模块位于 `backend/`
- 前端位于 `frontend/`
- 部署相关文件位于 `deploy/`
- 当前生产环境信息：
  - 主机：`root@luokyme.com`
  - 应用目录：`/opt/sub2api`
  - systemd 单元：`sub2api`
  - 运行配置：`/opt/sub2api/config.yaml`

## 工作约定

- 不要假设仓库根目录就是 Go module 根目录。运行 Go 命令时应先进入 `backend/`。
- 除非改动范围很大，否则优先跑定向测试，不要默认跑全量测试。
- `prompt_cache_key` 和真实会话标识必须分离处理，不要混用。
- 当前 compat 路径中的 `prompt_cache_key` 应只表达稳定前缀，不应包含首条用户消息。
- 对 OpenAI OAuth 上游，compat 自动注入的稳定 `prompt_cache_key` 应优先用于上游 `session_id` / `conversation_id`。
- 生产重启要谨慎：服务停止时会触发 Redis 清理，重启后 Redis 侧缓存会冷启动。
- 仅修改后端代码时，仍需使用 `-tags embed` 构建生产二进制；否则可能导致前端嵌入资源失效。

## 常用命令

### 后端

```bash
cd backend
go test ./internal/service/... ./internal/handler/...
```

构建后端二进制：

```bash
cd backend
VERSION=$(tr -d '\r\n' < ./cmd/server/VERSION)
go build -tags embed -ldflags "-s -w -X main.Version=$VERSION" -trimpath -o /tmp/sub2api-deploy ./cmd/server
```

### 前端

```bash
cd frontend
pnpm test
pnpm build
```

## 部署流程

以下流程对应当前 `luokyme.com` 生产机的二进制部署方式。

### 1. 本地构建

```bash
cd /home/lkm/ws/lkm/sub2api/backend
VERSION=$(tr -d '\r\n' < ./cmd/server/VERSION)
go build -tags embed -ldflags "-s -w -X main.Version=$VERSION" -trimpath -o /tmp/sub2api-deploy ./cmd/server
sha256sum /tmp/sub2api-deploy
```

如果前端资源有改动，必须先重新构建前端，确保 `backend/internal/web/dist/` 为最新：

```bash
cd /home/lkm/ws/lkm/sub2api/frontend
pnpm build
```

### 2. 上传二进制

为新文件选择一个有描述性的后缀：

```bash
scp /tmp/sub2api-deploy root@luokyme.com:/opt/sub2api/sub2api.new-<tag>
```

示例：

- `compat-session-fix-20260418T2348`
- `prefix-only-key-20260418T2359`
- `fix-foo-YYYYMMDDTHHMMSS`

### 3. 备份当前版本并切换

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

### 4. 验证服务状态

```bash
ssh root@luokyme.com '
systemctl is-active sub2api
systemctl status sub2api --no-pager -l | sed -n "1,25p"
journalctl -u sub2api -n 40 --no-pager
curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/
curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/index.html
'
```

最低检查标准：

- `systemctl is-active sub2api` 返回 `active`
- `http://127.0.0.1:8080/` 返回 `200`
- `http://127.0.0.1:8080/index.html` 返回 `200`

### 5. 必要时回滚

列出最近备份：

```bash
ssh root@luokyme.com 'cd /opt/sub2api && ls -1t sub2api.bak.* | head'
```

恢复指定备份：

```bash
ssh root@luokyme.com '
set -euo pipefail
cd /opt/sub2api
install -m 755 -o root -g root sub2api.bak.<timestamp>.pre-deploy sub2api
systemctl restart sub2api
'
```

## 运维说明

- 当前 systemd 单元文件路径：`/etc/systemd/system/sub2api.service`
- 当前还存在一个 drop-in：`/etc/systemd/system/sub2api.service.d/debug-gateway.conf`
- 该 drop-in 当前开启了 gateway body 调试日志：

```ini
Environment=SUB2API_DEBUG_GATEWAY_BODY=/opt/sub2api/gateway_debug.log
```

- 排查生产环境 prompt cache 行为时，重点查看：
  - `journalctl -u sub2api`
  - `/opt/sub2api/gateway_debug.log`
  - PostgreSQL 中的 `usage_logs`

## Prompt Cache 改动验收

如果改动涉及 compat 缓存、会话路由或上游 session 选择，至少验证以下几项：

1. 同一稳定前缀在后续轮次中 `prompt_cache_key` 保持不变。
2. 不同首条用户消息不会再导致 compat `prompt_cache_key` 分叉。
3. 不同稳定前缀仍然会生成不同的 `prompt_cache_key`。
4. compat 自动注入的稳定 key 会传递到上游 `session_id` / `conversation_id`。
5. `usage_logs` 中能看到符合预期的 `cache_read_tokens`。
