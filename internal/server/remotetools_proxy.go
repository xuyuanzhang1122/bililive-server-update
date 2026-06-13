package server

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// remotetools 下载中转：bililive-go 的 remote-tools-config.json 把
// https://image.xumy.art/remotetools/download?downloadurl=<github url>
// 作为工具下载兜底。仅允许白名单上游，防止被当作开放代理滥用。

var remotetoolsAllowedHosts = map[string]bool{
	"github.com":                           true,
	"objects.githubusercontent.com":        true,
	"release-assets.githubusercontent.com": true,
	"builds.dotnet.microsoft.com":          true,
}

var remotetoolsClient = &http.Client{
	Timeout: 30 * time.Minute,
}

func (s *Server) remotetoolsDownload(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("downloadurl"))
	if raw == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("缺少 downloadurl 参数"))
		return
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("downloadurl 必须是合法的 https 链接"))
		return
	}
	if !remotetoolsAllowedHosts[u.Hostname()] {
		writeError(w, http.StatusForbidden, fmt.Errorf("上游域名不在白名单内: %s", u.Hostname()))
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	req.Header.Set("User-Agent", "bililive-server-update-proxy")

	resp, err := remotetoolsClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("上游请求失败: %w", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, fmt.Errorf("上游返回 %s", resp.Status))
		return
	}

	for _, h := range []string{"Content-Type", "Content-Length", "Content-Disposition"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}
