package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuyuanzhang1122/bililive-server-update/internal/model"
)

// 源站制品管理：
//   - release 资产镜像：sync-github 时下载最新 Release 的全部资产到本地，仅保留最新版本
//   - 工具二进制分发：管理员手动上传 ffmpeg / 无头浏览器 等稳定版工具，不自动更新
//   - 静态下载：统一从 /artifacts/... 提供，catalog 中 URL 为相对路径，客户端拼接源站地址

const mirrorDownloadTimeout = 30 * time.Minute

// mirrorReleaseAssets 把 release 资产下载到 artifacts/releases/<version>/，
// 并把 catalog 条目的 URL 重写为本站相对路径。返回重写后的条目。
func (s *Server) mirrorReleaseAssets(ctx context.Context, items []model.ReleaseArtifact) ([]model.ReleaseArtifact, error) {
	if len(items) == 0 {
		return items, nil
	}
	version := items[0].Version
	versionDir := filepath.Join(s.store.ArtifactsDir(), "releases", sanitizePathComponent(version))
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return nil, err
	}

	dlCtx, cancel := context.WithTimeout(ctx, mirrorDownloadTimeout)
	defer cancel()

	mirrored := make([]model.ReleaseArtifact, 0, len(items))
	for _, item := range items {
		name := sanitizePathComponent(item.Name)
		if name == "" || item.URL == "" || !strings.HasPrefix(item.URL, "https://") {
			mirrored = append(mirrored, item)
			continue
		}
		dest := filepath.Join(versionDir, name)
		if err := downloadFile(dlCtx, item.URL, dest); err != nil {
			return nil, fmt.Errorf("下载 %s 失败: %w", item.Name, err)
		}
		if item.Metadata == nil {
			item.Metadata = map[string]string{}
		}
		item.Metadata["upstream_url"] = item.URL
		item.Source = "mirror"
		item.URL = "/artifacts/releases/" + sanitizePathComponent(version) + "/" + name
		mirrored = append(mirrored, item)
	}

	// 仅保留最新版本：清掉其他版本目录
	entries, err := os.ReadDir(filepath.Join(s.store.ArtifactsDir(), "releases"))
	if err == nil {
		for _, e := range entries {
			if e.IsDir() && e.Name() != sanitizePathComponent(version) {
				_ = os.RemoveAll(filepath.Join(s.store.ArtifactsDir(), "releases", e.Name()))
			}
		}
	}
	return mirrored, nil
}

// uploadTool 管理员上传稳定版工具二进制
// 路由：PUT /api/v1/tools/{name}?os=linux&arch=amd64&version=6.1&filename=ffmpeg.tar.gz
func (s *Server) uploadTool(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	name := sanitizePathComponent(r.PathValue("name"))
	osName := sanitizePathComponent(r.URL.Query().Get("os"))
	arch := sanitizePathComponent(r.URL.Query().Get("arch"))
	version := sanitizePathComponent(r.URL.Query().Get("version"))
	filename := sanitizePathComponent(r.URL.Query().Get("filename"))
	if name == "" || osName == "" || arch == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("name/os/arch 不能为空"))
		return
	}
	if version == "" {
		version = "stable"
	}
	if filename == "" {
		filename = fmt.Sprintf("%s-%s-%s", name, osName, arch)
	}

	toolDir := filepath.Join(s.store.ArtifactsDir(), "tools", name, osName+"-"+arch)
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 同平台旧文件清掉，工具目录里只保留一个稳定版
	if old, err := os.ReadDir(toolDir); err == nil {
		for _, e := range old {
			_ = os.Remove(filepath.Join(toolDir, e.Name()))
		}
	}
	dest := filepath.Join(toolDir, filename)
	f, err := os.Create(dest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	size, err := io.Copy(f, r.Body)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(dest)
		if err == nil {
			err = closeErr
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	relURL := "/artifacts/tools/" + name + "/" + osName + "-" + arch + "/" + filename
	now := time.Now().UTC()
	artifact := model.ToolArtifact{
		Name:      name,
		Version:   version,
		OS:        osName,
		Arch:      arch,
		URL:       relURL,
		Stable:    true,
		UpdatedAt: now,
	}

	catalog, err := s.store.LoadCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	replaced := false
	for i, t := range catalog.Tools {
		if t.Name == name && t.OS == osName && t.Arch == arch {
			catalog.Tools[i] = artifact
			replaced = true
			break
		}
	}
	if !replaced {
		catalog.Tools = append(catalog.Tools, artifact)
	}
	catalog.UpdatedAt = now
	if err := s.store.SaveCatalog(catalog); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"tool": artifact,
		"size": size,
	})
}

// serveArtifacts 静态提供镜像的制品文件
func (s *Server) serveArtifacts() http.Handler {
	fs := http.FileServer(http.Dir(s.store.ArtifactsDir()))
	return http.StripPrefix("/artifacts/", fs)
}

// refreshInstallScripts 从 GitHub raw 拉取最新安装脚本并缓存到本地
func (s *Server) refreshInstallScripts(ctx context.Context) error {
	base := "https://raw.githubusercontent.com/" + strings.Trim(s.cfg.GitHubRepo, "/") + "/main/scripts"
	for _, name := range []string{"install.sh", "install.ps1"} {
		dest := filepath.Join(s.store.ArtifactsDir(), name)
		if err := downloadFile(ctx, base+"/"+name, dest); err != nil {
			// install.ps1 允许暂缺
			if name == "install.ps1" {
				continue
			}
			return fmt.Errorf("拉取 %s 失败: %w", name, err)
		}
	}
	return nil
}

// serveInstallScript 提供缓存的安装脚本；本地无缓存时从 GitHub 现拉一次。
// 输出前把 DEFAULT_MIRROR 占位替换为本站地址，让脚本默认指向当前源站。
func (s *Server) serveInstallScript(w http.ResponseWriter, r *http.Request, name, contentType string) {
	path := filepath.Join(s.store.ArtifactsDir(), name)
	data, err := os.ReadFile(path)
	if err != nil {
		if err := s.refreshInstallScripts(r.Context()); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		data, err = os.ReadFile(path)
		if err != nil {
			writeError(w, http.StatusNotFound, fmt.Errorf("安装脚本不可用: %w", err))
			return
		}
	}
	base := publicBaseURL(s.cfg, r)
	out := strings.Replace(string(data), `DEFAULT_MIRROR=""`, `DEFAULT_MIRROR="`+base+`"`, 1)
	out = strings.Replace(out, `$DefaultMirror = ""`, `$DefaultMirror = "`+base+`"`, 1)
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write([]byte(out))
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "bililive-server-update")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("下载 %s 返回 %s", url, resp.Status)
	}
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func sanitizePathComponent(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, "..", "")
	v = strings.ReplaceAll(v, "/", "")
	v = strings.ReplaceAll(v, "\\", "")
	return v
}
