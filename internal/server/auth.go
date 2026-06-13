package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 管理面板登录鉴权：
//   - 凭据用常量时间比较，避免计时侧信道
//   - 登录成功签发 HMAC-SHA256 签名的会话 cookie（HttpOnly），不在 cookie 里放任何明文凭据
//   - 会话密钥由 admin token + 密码派生，重启后仍有效，但泄露 cookie 无法反推密码
//   - 管理 API 同时接受 Bearer admin token（脚本/自动化用）或有效会话 cookie（浏览器用）

const (
	sessionCookieName = "blsu_session"
	sessionTTL        = 7 * 24 * time.Hour
)

func (s *Server) sessionSecret() []byte {
	h := sha256.Sum256([]byte(s.cfg.AdminToken + ":" + s.cfg.AdminPassword + ":blsu-session-v1"))
	return h[:]
}

// signSession 生成 "<expiryUnix>.<base64(hmac)>" 形式的会话令牌
func (s *Server) signSession(expiry int64) string {
	payload := strconv.FormatInt(expiry, 10)
	mac := hmac.New(sha256.New, s.sessionSecret())
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig
}

// verifySession 校验会话令牌签名与有效期
func (s *Server) verifySession(token string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	expiry, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return false
	}
	mac := hmac.New(sha256.New, s.sessionSecret())
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) == 1
}

func (s *Server) hasValidSession(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return s.verifySession(c.Value)
}

// adminLogin 处理登录请求
func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	userOK := subtle.ConstantTimeCompare([]byte(req.Username), []byte(s.cfg.AdminUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(req.Password), []byte(s.cfg.AdminPassword)) == 1
	if !userOK || !passOK {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("用户名或密码错误"))
		return
	}
	expiry := time.Now().Add(sessionTTL).Unix()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    s.signSession(expiry),
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(expiry, 0),
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": s.cfg.AdminUser})
}

// adminLogout 清除会话 cookie
func (s *Server) adminLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// adminMe 返回当前登录状态（管理页用于判断是否展示登录框）
func (s *Server) adminMe(w http.ResponseWriter, r *http.Request) {
	if s.hasValidSession(r) {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "user": s.cfg.AdminUser})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
}
