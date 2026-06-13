package server

import (
	"embed"
	"net/http"
)

//go:embed webui/index.html webui/landing.html
var webuiFS embed.FS

// serveLanding 提供公开的三项目介绍落地页
func (s *Server) serveLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.serveEmbedded(w, "webui/landing.html")
}

// serveAdmin 提供管理界面（页面自身通过 /api/admin/me 判断登录态）
func (s *Server) serveAdmin(w http.ResponseWriter, _ *http.Request) {
	s.serveEmbedded(w, "webui/index.html")
}

func (s *Server) serveEmbedded(w http.ResponseWriter, name string) {
	data, err := webuiFS.ReadFile(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// listBackups 管理界面备份列表（仅摘要，不含包体）
func (s *Server) listBackups(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	list, err := s.store.ListBackups()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
