# 统一安装部署流程

目标：统一为一个入口，由安装器根据系统和用户选择完成安装。

## 1. 入口命令

macOS / Linux：

```bash
curl -fsSL https://update.example.com/install.sh | sh
```

Windows PowerShell：

```powershell
curl.exe -fsSL https://update.example.com/install.ps1 -o install.ps1; powershell -ExecutionPolicy Bypass -File .\install.ps1
```

## 2. 安装器流程

1. 检测系统：Windows / macOS / Linux、CPU 架构、权限。
2. 检测旧版本数据位置：
   - 只针对视频目录复用。
   - 如果用户选择旧目录，`out_put_path` 指向旧视频目录。
   - `app_data_path` 仍建议使用新版本独立 Data 目录。
3. 选择源：
   - 自建源：读取 `bililive-server-update` catalog。
   - GitHub 源：直接读取 GitHub Release。
4. 检测工具：
   - ffmpeg：配置路径优先，然后环境变量/PATH，再从自建源下载。
   - headless-browser：配置路径优先，然后环境变量/PATH，再从自建源下载。
5. 选择安装方式：
   - binary：默认。
   - docker：需要创建 Videos/Data/config.yml 挂载。
   - npm：如果最终只是调用 binary/docker，应下线。
6. 确认目录和端口：
   - install dir
   - output path
   - app data path
   - port
7. 下载并安装。
8. 调用 doctor：
   - 本项目 `POST /api/v1/doctor` 用于统一云端判断格式。
   - 用户机器上的主服务也应提供 `POST /api/local/doctor` 做真实本机检查。
9. 显示服务状态：
   - Web UI 地址
   - API 地址
   - 配置路径
   - 视频目录
   - Data 目录
   - ffmpeg/headless-browser 状态

## 3. 自建源维护规则

Release 源：

- 自动同步 GitHub latest release。
- 自建源只保留最新版本。
- Docker/npm 如果继续保留，也只保留最新 tag 或 latest metadata。

工具源：

- ffmpeg、headless-browser 等工具使用稳定版本。
- 不主动跟随上游更新。
- 只有管理员触发 `PUT /api/v1/catalog/tools` 后才更新。

