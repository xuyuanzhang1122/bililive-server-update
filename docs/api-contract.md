# API 契约

本文面向 iOS、Web 前端、`bililive-go-UI` 后端。分为三类：

- **复用主服务**：已经在 `bililive-go-UI` 中存在或刚合入，客户端应直接使用。
- **主服务需新增**：不属于本项目，但服务器端后续必须补齐。
- **本项目提供**：`bililive-server-update` 已实现的接口。

## 1. 复用主服务 API

### 认证与 Key 用户

#### `GET /api/auth/me`

用途：iOS 绑定 API Key 后验证当前 Key 属于哪个用户。

请求头：

```http
Authorization: Bearer blgo_xxx
```

响应：

```json
{
  "id": "usr_abcd1234",
  "name": "iPhone 15 Pro",
  "key_suffix": "A1B2",
  "enabled": true,
  "last_used_at": "2026-05-04 18:30:00"
}
```

#### `GET /api/api-keys`

用途：Web 设置页展示所有 Key 用户，不返回完整 Key。

#### `POST /api/api-keys`

用途：Web 设置页创建 Key 用户。完整 Key 只展示一次。

请求：

```json
{ "name": "iPhone 15 Pro" }
```

响应：

```json
{
  "id": "usr_xxx",
  "name": "iPhone 15 Pro",
  "api_key": "blgo_xxx",
  "key_suffix": "1234",
  "enabled": true
}
```

#### `PATCH /api/api-keys/{id}`

用途：改名、启用、禁用。

```json
{ "name": "iPad", "enabled": true }
```

#### `DELETE /api/api-keys/{id}`

用途：吊销 Key。

### 观看历史

#### `GET /api/history`

用途：iOS/Web 获取当前 Key 用户的观看历史。

响应：

```json
[
  {
    "id": 1,
    "api_key_user_id": "usr_xxx",
    "video_path": "抖音/主播/a.flv",
    "video_name": "a.flv",
    "position_seconds": 95.2,
    "duration_seconds": 600.0,
    "updated_at": "2026-05-04 18:30:00"
  }
]
```

#### `GET /api/history/{videoPath}`

用途：播放器打开前读取单个视频的续播点。`videoPath` 需要逐段 URL encode。

#### `POST /api/history`

用途：播放器定时保存进度。Web 端也必须接入此接口，不能只存 localStorage。

```json
{
  "video_path": "抖音/主播/a.flv",
  "video_name": "a.flv",
  "position_seconds": 95.2,
  "duration_seconds": 600.0
}
```

#### `DELETE /api/history/{videoPath}`

用途：删除当前 Key 用户的单条历史。

### 视频播放

#### `GET /api/video-library`

用途：视频库首页。继续复用。

客户端注意：

- `recording=true` 展示直播中标识
- 主服务必须修复直播结束后仍为 `true` 的问题

#### `GET /api/video-files/{folderPath}`

用途：文件列表、播放 URL、缩略图 URL。继续复用。

现有字段：

```json
{
  "name": "a.flv",
  "rel_path": "抖音/主播/a.flv",
  "size": 1024,
  "mod_time": 1777900000,
  "file_url": "/files/...",
  "thumbnail_url": "/api/thumbnail/...",
  "hls_url": "/api/stream/hls/..."
}
```

Web/iOS 录制中文件策略：

- 有 `hls_url` 或原生可播 `file_url`：直接打开播放器
- 打开失败或主服务返回 `playback_status=recording`：留在当前页提示“正在录制请稍后”
- 不再跳外部页面

#### `GET /api/thumbnail/{videoPath}`

用途：缩略图。iOS 显示必须使用等比填充或等比适配，避免拉伸。

#### `GET /api/stream/hls/{videoPath}`

用途：FLV/TS/MKV 转 HLS 给 iOS AVPlayer/Pillarbox 播放。

### 短链解析

#### `GET /api/resolve-url?url=...`

用途：解析分享文案、短链、标准直播间 URL。继续复用。

主服务后续应扩展无头浏览器解析链路。

## 2. 主服务需新增 API

### 无头浏览器配置

#### `GET /api/config/headless-browser`

```json
{
  "path": "/opt/bililive-tools/chromium",
  "auto_install": true,
  "timeout_seconds": 15,
  "detected_path": "/usr/bin/chromium",
  "available": true,
  "version": "Chromium 123"
}
```

#### `PATCH /api/config/headless-browser`

```json
{
  "path": "/opt/bililive-tools/chromium",
  "auto_install": true,
  "timeout_seconds": 15
}
```

#### `POST /api/tools/headless-browser/probe`

用途：Web 设置页点击检测。

### Douyin Cookie

#### `GET /api/config/douyin-cookie`

响应不应直接回完整 cookie，只回是否已配置和更新时间。

```json
{
  "configured": true,
  "updated_at": "2026-05-04 18:30:00"
}
```

#### `PUT /api/config/douyin-cookie`

```json
{ "cookie": "sessionid=..." }
```

### 本机恢复与安装 doctor

这些接口属于用户机器上的 `bililive-go-UI` 或本机管理工具，不属于本项目。

#### `POST /api/local/doctor`

输入同本项目 `POST /api/v1/doctor`。

#### `POST /api/local/restore-config`

```json
{
  "backup_id": "abcd1234",
  "server": {
    "rpc_bind": ":8080",
    "port": 8080,
    "output_path": "/srv/bililive",
    "app_data_path": "/var/lib/bililive"
  },
  "rooms": [
    { "url": "https://live.douyin.com/xxx", "is_listening": true }
  ],
  "restart": true
}
```

#### `POST /api/local/restart`

用途：配置恢复后重启服务。

## 3. 本项目提供 API

默认 base URL 示例：`https://update.example.com`。

### `GET /health`

```json
{ "ok": true, "service": "bililive-server-update" }
```

### `GET /api/v1/install/manifest`

用途：统一安装器第一步拉取清单。

响应包含：

- 可选源：自建源、GitHub
- 安装方式：binary、docker、npm
- 必需工具：ffmpeg、headless-browser
- doctor API 地址
- backup API 地址
- 当前 catalog

### `POST /api/v1/doctor`

用途：安装器把本机检测结果发给服务器，服务器返回统一判断。

请求：

```json
{
  "os": "linux",
  "arch": "amd64",
  "install_mode": "binary",
  "port": 8080,
  "paths": {
    "output_path": "/srv/bililive",
    "app_data_path": "/var/lib/bililive"
  },
  "tools": [
    { "id": "ffmpeg", "path": "/usr/bin/ffmpeg", "version": "6.1", "ok": true },
    { "id": "headless-browser", "path": "", "ok": false }
  ]
}
```

响应：

```json
{
  "ok": true,
  "checks": [
    { "id": "ffmpeg", "ok": true, "message": "ffmpeg 需要可执行路径" }
  ]
}
```

### `GET /api/v1/catalog`

用途：安装器或管理后台读取发布源与工具源。

### `POST /api/v1/catalog/sync-github`

用途：手动触发从 GitHub latest release 同步到 catalog，仅保留当前 GitHub latest。

需要：

```http
Authorization: Bearer <BLSU_ADMIN_TOKEN>
```

### `PUT /api/v1/catalog/releases`

用途：管理后台直接覆盖 release 清单。需要 admin token。

### `PUT /api/v1/catalog/tools`

用途：管理后台直接覆盖稳定工具清单。需要 admin token。

### `POST /api/v1/backups`

用途：iOS 一键备份上传到你的服务器。

请求：

```json
{
  "device_name": "iPhone 15 Pro",
  "app_version": "1.1.0",
  "server": {
    "base_url": "http://192.168.1.2:8080",
    "rpc_bind": ":8080",
    "port": 8080,
    "output_path": "/srv/bililive",
    "app_data_path": "/var/lib/bililive",
    "config_path": "/etc/bililive-go/config.yml"
  },
  "rooms": [
    { "url": "https://live.douyin.com/xxx", "is_listening": true, "scheme": "" }
  ],
  "package_base64": "optional-client-side-archive",
  "client_hash": "sha256..."
}
```

响应：

```json
{
  "id": "k5m2abc9",
  "created_at": "2026-05-04T10:30:00Z",
  "schema_version": 1,
  "server": {},
  "rooms": []
}
```

iOS 需要把 `id` 展示给用户。

### `GET /api/v1/backups/{id}`

用途：iOS 输入 ID 找回配置。

### `POST /api/v1/backups/{id}/restore-request`

用途：生成恢复计划。真正写配置由用户机器上的本机管理接口执行。

请求：

```json
{ "target_server_url": "http://192.168.1.2:8080", "dry_run": true }
```

响应：

```json
{
  "backup_id": "k5m2abc9",
  "dry_run": true,
  "requires_local": true,
  "local_tool_api": "/api/local/restore-config",
  "steps": ["下载备份并校验", "写入 config.yml", "运行 doctor", "重启服务"]
}
```

