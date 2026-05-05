# iOS / Web 改动清单

## iOS

### 设置页

- 使用 `GET /api/auth/me` 验证 API Key。
- 绑定成功后展示 Key 用户名称、ID、尾号。
- 清空 Key 时必须从 Keychain 删除，而不是只清内存。

### 视频列表

- 使用 `GET /api/video-files/{folder}` 获取播放 URL。
- 同时使用 `GET /api/history` 映射 `video_path -> position/duration`。
- 缩略图显示必须保持比例，避免把 16:9 拉成固定正方形。

### 播放页

- 打开视频后先调用 `GET /api/history/{videoPath}`。
- 如果 `position_seconds` 在 `(5, duration-5)`，seek 到该位置后播放。
- 定时调用 `POST /api/history` 同步进度。
- 进度条需要支持点击跳转。
- 长按侧边进入 2x 临时加速：
  - 长按时隐藏控制条。
  - 底部显示“下拉锁定二倍速”。
  - 手指移动到下三分之一处时锁定 2x。
  - 再次长按时显示“下拉解锁二倍速”。
  - 再次移动到下三分之一处恢复 1x。

### 直播间备份

- 从当前服务器读取：
  - `GET /api/config` 或 `GET /api/raw-config`
  - `GET /api/lives`
- 本地生成一份可分享文件。
- 同时 `POST /api/backups` 上传到远端备份服务或当前主服务。
- 展示返回的 `id`。
- 找回时：
  - 输入 ID 调 `GET /api/backups/{id}`。
  - 上传本地备份文件时直接解析。
  - 调用户服务器的 `POST /api/backups/restore` 执行恢复。
  - 如果返回 `job_id`，轮询 `GET /api/backups/restore/status/{job_id}`。
  - 轮询 `/api/info` 和 `/api/lives` 等待重启完成。

## Web

### 设置页

- API Key 管理继续使用：
  - `GET /api/api-keys`
  - `POST /api/api-keys`
  - `PATCH /api/api-keys/{id}`
  - `DELETE /api/api-keys/{id}`
- 新增二级菜单：
  - 无头浏览器路径、自动安装、检测按钮
  - Douyin Cookie 配置

### 视频库/播放器

- 录制中文件点击后不跳外部页面。
- 有可播放 URL 时内嵌播放。
- 无法播放时显示“正在录制请稍后”。
- Web 端必须接入 `POST /api/history`，同步观看进度到服务端，而不是只写 localStorage。
- 播放器打开时使用 `GET /api/history/{videoPath}` 续播。

## 主后端

- 修复直播结束后 `/api/video-library` 的 `recording` 不自动变 false。
- 给短链解析接入无头浏览器 fallback。
- 添加 headless browser 和 Douyin Cookie 配置 API。
- 添加本机 doctor/restart API，供安装器和恢复流程检测。
- 添加 `/api/backups/restore` 与 restore status，供 iOS 找回配置后自动恢复。

当前已检查分支：

- Web/主服务：`feature/web-main-service-integration`
- iOS：`fix/ios-playback-sync-backup`

这两个分支的备份恢复路径已经统一为 `/api/backups...`，本项目保留 `/api/v1/backups...` 作为更通用的源服务器 API，同时兼容 `/api/backups...` 给 iOS 当前实现使用。
