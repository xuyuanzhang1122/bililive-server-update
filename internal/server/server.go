package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xuyuanzhang1122/bililive-server-update/internal/config"
	"github.com/xuyuanzhang1122/bililive-server-update/internal/model"
	"github.com/xuyuanzhang1122/bililive-server-update/internal/store"
)

type Server struct {
	cfg   config.Config
	store *store.FileStore
	mux   *http.ServeMux
}

func New(cfg config.Config, fileStore *store.FileStore) http.Handler {
	s := &Server{
		cfg:   cfg,
		store: fileStore,
		mux:   http.NewServeMux(),
	}
	s.routes()
	return withCORS(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.serveLanding)
	s.mux.HandleFunc("GET /admin", s.serveAdmin)
	s.mux.HandleFunc("POST /api/admin/login", s.adminLogin)
	s.mux.HandleFunc("POST /api/admin/logout", s.adminLogout)
	s.mux.HandleFunc("GET /api/admin/me", s.adminMe)
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("GET /install.sh", s.installShell)
	s.mux.HandleFunc("GET /install.ps1", s.installPowerShell)
	s.mux.HandleFunc("GET /api/v1/install/manifest", s.installManifest)
	s.mux.HandleFunc("POST /api/v1/doctor", s.doctor)
	s.mux.HandleFunc("GET /api/v1/catalog", s.getCatalog)
	s.mux.HandleFunc("POST /api/v1/catalog/sync-github", s.syncGitHubCatalog)
	s.mux.HandleFunc("PUT /api/v1/catalog/releases", s.replaceReleases)
	s.mux.HandleFunc("PUT /api/v1/catalog/tools", s.replaceTools)
	s.mux.HandleFunc("PUT /api/v1/tools/{name}", s.uploadTool)
	s.mux.Handle("GET /artifacts/", s.serveArtifacts())
	s.mux.HandleFunc("GET /remotetools/download", s.remotetoolsDownload)
	s.mux.HandleFunc("GET /api/v1/backups", s.listBackups)
	s.mux.HandleFunc("POST /api/v1/backups", s.createBackup)
	s.mux.HandleFunc("GET /api/v1/backups/{id}", s.getBackup)
	s.mux.HandleFunc("POST /api/v1/backups/{id}/restore-request", s.restoreRequest)
	s.mux.HandleFunc("POST /api/backups", s.createIOSBackup)
	s.mux.HandleFunc("GET /api/backups/{id}", s.getIOSBackup)
	s.mux.HandleFunc("POST /api/backups/restore", s.restoreIOSBackup)
	s.mux.HandleFunc("GET /api/backups/restore/status/{job_id}", s.restoreIOSBackupStatus)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"service":    "bililive-server-update",
		"updated_at": time.Now().UTC(),
	})
}

func (s *Server) getCatalog(w http.ResponseWriter, _ *http.Request) {
	catalog, err := s.store.LoadCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, catalog)
}

func (s *Server) installManifest(w http.ResponseWriter, r *http.Request) {
	catalog, err := s.store.LoadCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	base := publicBaseURL(s.cfg, r)
	writeJSON(w, http.StatusOK, model.InstallManifest{
		PublicBaseURL: base,
		GeneratedAt:   time.Now().UTC(),
		Sources: []model.InstallSource{
			{ID: "mirror", Name: "自建源", Description: "优先使用 bililive-server-update 管理的镜像源", Default: true},
			{ID: "github", Name: "GitHub", Description: "直接使用 GitHub Release / npm / Docker 官方上游"},
		},
		Modes: []model.InstallMode{
			{ID: "binary", Name: "二进制 Release", Description: "默认安装方式，适合裸机运行"},
			{ID: "docker", Name: "Docker", Description: "容器运行，需挂载 Videos/Data/config.yml"},
			{ID: "npm", Name: "npm 启动器", Description: "仅保留兼容入口；若最终只是拉 Docker 或二进制，可下线", Deprecated: true},
		},
		RequiredTools: []model.RequiredTool{
			{
				ID:          "ffmpeg",
				Name:        "FFmpeg",
				Required:    true,
				ConfigKey:   "ffmpeg_path",
				EnvKeys:     []string{"FFMPEG_PATH", "PATH"},
				Description: "录制、缩略图、HLS 转封装和后处理依赖",
			},
			{
				ID:          "headless-browser",
				Name:        "Headless Browser",
				Required:    false,
				ConfigKey:   "headless_browser.path",
				EnvKeys:     []string{"BILILIVE_HEADLESS_BROWSER_PATH", "PLAYWRIGHT_BROWSERS_PATH", "PATH"},
				Description: "用于解析不标准短链或 JS 跳转",
			},
		},
		DoctorAPI: "/api/v1/doctor",
		BackupAPI: "/api/v1/backups",
		Catalog:   catalog,
	})
}

func (s *Server) doctor(w http.ResponseWriter, r *http.Request) {
	var req model.DoctorRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	checks := []model.DoctorCheck{
		{ID: "os", OK: req.OS != "", Message: nonEmptyMessage(req.OS, "系统类型已上报", "缺少系统类型")},
		{ID: "arch", OK: req.Arch != "", Message: nonEmptyMessage(req.Arch, "CPU 架构已上报", "缺少 CPU 架构")},
		{ID: "install_mode", OK: req.InstallMode != "", Message: nonEmptyMessage(req.InstallMode, "安装方式已选择", "缺少安装方式")},
		{ID: "port", OK: req.Port > 0 && req.Port < 65536, Message: "端口需要在 1-65535 之间"},
		{ID: "output_path", OK: strings.TrimSpace(req.Paths["output_path"]) != "", Message: "需要确认录播输出目录"},
	}
	toolOK := map[string]bool{}
	for _, tool := range req.Tools {
		toolOK[tool.ID] = tool.OK && strings.TrimSpace(tool.Path) != ""
	}
	checks = append(checks,
		model.DoctorCheck{ID: "ffmpeg", OK: toolOK["ffmpeg"], Message: "ffmpeg 需要可执行路径"},
		model.DoctorCheck{ID: "headless-browser", OK: toolOK["headless-browser"], Message: "无头浏览器可缺省；短链 JS 跳转解析会降级"},
	)
	ok := true
	for _, check := range checks {
		if !check.OK && check.ID != "headless-browser" {
			ok = false
			break
		}
	}
	writeJSON(w, http.StatusOK, model.DoctorResponse{OK: ok, Checks: checks})
}

func (s *Server) syncGitHubCatalog(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	release, err := fetchLatestGitHubRelease(r.Context(), s.cfg.GitHubRepo)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	// 默认把资产镜像到本站（仅保留最新版本），?mirror=false 时只更新元数据
	if r.URL.Query().Get("mirror") != "false" {
		mirrored, err := s.mirrorReleaseAssets(r.Context(), release)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		release = mirrored
	}
	// 顺带刷新安装脚本缓存，保证 /install.sh 提供的是仓库最新版
	if err := s.refreshInstallScripts(r.Context()); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	catalog, err := s.store.LoadCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	next := catalog.Releases[:0]
	for _, item := range catalog.Releases {
		if item.Source != "github" && item.Source != "mirror" {
			next = append(next, item)
		}
	}
	next = append(next, release...)
	catalog.Releases = next
	catalog.UpdatedAt = time.Now().UTC()
	if err := s.store.SaveCatalog(catalog); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, catalog)
}

func (s *Server) replaceReleases(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var releases []model.ReleaseArtifact
	if err := decodeJSON(r.Body, &releases); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	now := time.Now().UTC()
	for i := range releases {
		if releases[i].UpdatedAt.IsZero() {
			releases[i].UpdatedAt = now
		}
	}
	catalog, err := s.store.LoadCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	catalog.Releases = releases
	catalog.UpdatedAt = now
	if err := s.store.SaveCatalog(catalog); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, catalog)
}

func (s *Server) replaceTools(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var tools []model.ToolArtifact
	if err := decodeJSON(r.Body, &tools); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	now := time.Now().UTC()
	for i := range tools {
		if tools[i].UpdatedAt.IsZero() {
			tools[i].UpdatedAt = now
		}
	}
	catalog, err := s.store.LoadCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	catalog.Tools = tools
	catalog.UpdatedAt = now
	if err := s.store.SaveCatalog(catalog); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, catalog)
}

func (s *Server) createBackup(w http.ResponseWriter, r *http.Request) {
	var bundle model.BackupBundle
	if err := decodeJSON(r.Body, &bundle); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	saved, err := s.store.SaveBackup(bundle)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *Server) getBackup(w http.ResponseWriter, r *http.Request) {
	bundle, err := s.store.LoadBackup(r.PathValue("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, bundle)
}

func (s *Server) restoreRequest(w http.ResponseWriter, r *http.Request) {
	bundle, err := s.store.LoadBackup(r.PathValue("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	var req model.RestoreRequest
	if r.Body != nil {
		_ = decodeJSON(r.Body, &req)
	}
	target := req.TargetServerURL
	if target == "" {
		target = bundle.Server.BaseURL
	}
	writeJSON(w, http.StatusOK, model.RestorePlan{
		BackupID:        bundle.ID,
		DryRun:          req.DryRun,
		RequiresLocal:   true,
		LocalToolAPI:    "/api/local/restore-config",
		TargetServerURL: target,
		Steps: []string{
			"下载备份并校验 client_hash 或 package_base64",
			"由本机 bililive-go-UI 管理工具写入 config.yml",
			"恢复 output_path/app_data_path/rpc.bind 和 live_rooms",
			"运行 doctor 检查端口、目录、ffmpeg、无头浏览器",
			"检查通过后重启 bililive-go-UI 服务",
			"iOS/Web 轮询 /api/info 和 /api/lives 完成双端同步",
		},
	})
}

func (s *Server) createIOSBackup(w http.ResponseWriter, r *http.Request) {
	var pkg model.IOSBackupPackage
	if err := decodeJSON(r.Body, &pkg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id, createdAt, err := s.store.SaveIOSBackup(pkg)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, model.BackupCreateResponse{
		ID:        id,
		CreatedAt: createdAt,
	})
}

func (s *Server) getIOSBackup(w http.ResponseWriter, r *http.Request) {
	pkg, err := s.store.LoadIOSBackup(r.PathValue("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

func (s *Server) restoreIOSBackup(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
		"status":  "failed",
		"message": "此公网源服务只负责存取备份；写入 config.yml、重启 bililive-go 必须由当前 bililive-go 服务器实现 /api/backups/restore",
	})
}

func (s *Server) restoreIOSBackupStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, map[string]string{
		"status":  "failed",
		"job_id":  r.PathValue("job_id"),
		"message": "此服务不执行本机恢复任务，因此没有 restore job 状态",
	})
}

func (s *Server) installShell(w http.ResponseWriter, r *http.Request) {
	s.serveInstallScript(w, r, "install.sh", "text/x-shellscript; charset=utf-8")
}

func (s *Server) installPowerShell(w http.ResponseWriter, r *http.Request) {
	s.serveInstallScript(w, r, "install.ps1", "text/plain; charset=utf-8")
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	// 浏览器走登录会话 cookie
	if s.hasValidSession(r) {
		return true
	}
	// 脚本 / 自动化走 Bearer admin token
	if strings.TrimSpace(s.cfg.AdminToken) != "" {
		const prefix = "Bearer "
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, prefix) && strings.TrimSpace(strings.TrimPrefix(auth, prefix)) == s.cfg.AdminToken {
			return true
		}
	}
	writeError(w, http.StatusUnauthorized, fmt.Errorf("需要登录或有效的 admin token"))
	return false
}

func publicBaseURL(cfg config.Config, r *http.Request) string {
	if cfg.PublicBaseURL != "" {
		return strings.TrimRight(cfg.PublicBaseURL, "/")
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func nonEmptyMessage(value, okMessage, badMessage string) string {
	if strings.TrimSpace(value) == "" {
		return badMessage
	}
	return okMessage
}

func decodeJSON(r io.Reader, target any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{
		"error": err.Error(),
		"code":  status,
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
