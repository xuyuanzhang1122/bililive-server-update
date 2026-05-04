# 服务器端改造规划

本文只描述服务器端职责。iOS/Web 的交互改动见 `docs/api-contract.md`。

## 1. 服务拆分

### A. `bililive-go-UI` 主服务，优先复用

继续负责录播主业务：

- 直播间管理
- 直播状态、录制状态
- 视频库、视频文件列表
- 缩略图
- HLS 转封装播放
- 观看历史同步
- API Key 用户鉴权
- 短链解析
- 本机配置读写、doctor、重启服务

### B. `bililive-server-update` 新服务，本项目实现

负责外部源与备份：

- 给统一安装器提供安装 manifest
- 管理 GitHub/自建源/Docker/npm/二进制 release 元数据
- 管理 ffmpeg、无头浏览器等稳定工具版本
- 接收 iOS 上传的服务器配置与直播间备份
- 按短 ID 找回备份
- 生成恢复计划，但不直接操作用户机器文件

## 2. 附录需求对应关系

| 需求 | 服务端处理 |
| --- | --- |
| Web/iOS 录制中不跳转，能播放就播，不能播放提示稍后 | 复用 `GET /api/video-files/{folder}` 返回的 `file_url/hls_url`。主服务需在录制中文件不可播放时返回明确状态，建议新增 `playback_status` |
| 直播中标识结束后不自动转变 | 主服务修复 `livestate`/SSE/`/api/video-library` 的 recording 标识刷新 |
| 观看历史同步、续播 | 已复用主服务 `/api/history`、`/api/history/{videoPath}`，按 API Key 用户隔离 |
| iOS 缩略图变形 | 主服务缩略图 API 可复用，但 iOS 端显示需保持 aspect ratio；必要时主服务返回宽高 |
| iOS 进度条点击跳转 | 客户端改动，无新增服务端 |
| iOS 长按二倍速锁定/解锁 | 客户端手势改动，无新增服务端 |
| 短链解析 + 无头浏览器配置 + Douyin Cookie | 主服务需要新增/扩展配置 API；本项目只提供工具源清单 |
| 统一 curl 安装部署 | 本项目提供 manifest、catalog、doctor 响应格式、安装脚本入口 |
| iOS 导出/备份/找回 | 本项目实现备份上传、下载、恢复计划；主服务需要本机恢复 API/tool 执行写配置和重启 |

## 3. 主服务建议新增字段

### `GET /api/video-files/{folder}`

现有字段继续保留，建议补：

```json
{
  "name": "xxx.flv",
  "rel_path": "抖音/主播/xxx.flv",
  "file_url": "/files/...",
  "hls_url": "/api/stream/hls/...",
  "thumbnail_url": "/api/thumbnail/...",
  "recording": true,
  "playback_status": "ready | recording | processing | unsupported",
  "duration_seconds": 123.4,
  "width": 1920,
  "height": 1080
}
```

如果 `playback_status=recording` 且当前文件无法安全播放，Web/iOS 显示“正在录制请稍后”。

### `GET /api/video-library`

继续保留现有字段，必须保证直播结束后 `recording=false` 能自动刷新。建议由 `LiveEnd` 事件同步更新持久状态，SSE 推送 `video_library_changed`。

### 无头浏览器配置

建议在主服务 `config.yml` 新增：

```yaml
headless_browser:
  path: ""
  auto_install: true
  timeout_seconds: 15
douyin:
  cookie: ""
```

建议 API：

- `GET /api/config/headless-browser`
- `PATCH /api/config/headless-browser`
- `GET /api/config/douyin-cookie`
- `PUT /api/config/douyin-cookie`
- `POST /api/tools/headless-browser/probe`

## 4. 本机恢复工具要求

iOS 从本项目拿到备份后，真正写 `config.yml`、改端口、恢复 `live_rooms`、重启服务，必须发生在用户自己的服务器上。

主服务建议新增本机受保护接口：

- `POST /api/local/restore-config`
- `POST /api/local/doctor`
- `POST /api/local/restart`

这些接口必须要求管理员 Key，且只能在本机或同源 Web UI 中启用。

