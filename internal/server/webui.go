package server

import (
	"embed"
	"net/http"
)

//go:embed webui/index.html
var webuiFS embed.FS

// serveWebUI 提供内嵌的管理界面
func (s *Server) serveWebUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := webuiFS.ReadFile("webui/index.html")
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
