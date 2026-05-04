# bililive-server-update

`bililive-server-update` 是给 `bililive-go-UI` 生态配套的独立服务端，负责：

- 管理自建源的发布版本、Docker/npm/二进制安装入口元数据
- 管理 ffmpeg、无头浏览器等工具的稳定版本清单
- 为统一安装器提供 manifest 和 doctor 检测结果格式
- 存储 iOS/Web 上传的服务器与直播间配置备份，并通过短 ID 找回

它不替代 `bililive-go-UI` 的录播 API。录播、视频库、观看历史、HLS 等能力优先复用主项目已有接口；缺口已经记录在 [docs/api-contract.md](/Users/xu/Documents/GitHub/bililive-server-update/docs/api-contract.md)。

## 快速运行

```bash
go run ./cmd/server
```

默认监听 `:8090`，数据写入 `./data`。可用环境变量：

```bash
BLSU_ADDR=:8090
BLSU_DATA_DIR=./data
BLSU_ADMIN_TOKEN=change-me
BLSU_PUBLIC_BASE_URL=https://update.example.com
BLSU_GITHUB_REPO=xuyuanzhang1122/bililive-go-UI
```

## 主要接口

- `GET /health`
- `GET /api/v1/install/manifest`
- `POST /api/v1/doctor`
- `GET /api/v1/catalog`
- `POST /api/v1/catalog/sync-github`，需要 `Authorization: Bearer <admin-token>`
- `PUT /api/v1/catalog/releases`，需要 admin token
- `PUT /api/v1/catalog/tools`，需要 admin token
- `POST /api/v1/backups`
- `GET /api/v1/backups/{id}`
- `POST /api/v1/backups/{id}/restore-request`

完整请求/响应见 [docs/api-contract.md](/Users/xu/Documents/GitHub/bililive-server-update/docs/api-contract.md)。

