// Package gate protects the production dashboard and API with one shared password.
// It has no database, queue or provider access and is not part of local Compose.
package gate

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	cookieName  = "__Host-flowkit_session"
	sessionTTL  = 12 * time.Hour
	loginWindow = time.Minute
	loginLimit  = 5
	maxClients  = 4096
)

type Config struct {
	PasswordHash string
	SessionKey   string // Persisted, random 32-byte hex secret, independent of the password.
	Upstream     string
}

type loginCounter struct {
	start time.Time
	count int
}

type Gate struct {
	hash    []byte
	key     []byte
	proxy   *httputil.ReverseProxy
	log     *slog.Logger
	now     func() time.Time
	mu      sync.Mutex
	clients map[string]loginCounter
}

func New(cfg Config, log *slog.Logger) (*Gate, error) {
	if _, err := bcrypt.Cost([]byte(cfg.PasswordHash)); err != nil {
		return nil, fmt.Errorf("gate requires a valid DASHBOARD_PASSWORD_HASH")
	}
	secret, err := hex.DecodeString(cfg.SessionKey)
	if err != nil || len(secret) != 32 {
		return nil, fmt.Errorf("gate requires a random 32-byte hex DASHBOARD_SESSION_KEY")
	}
	upstream, err := url.Parse(cfg.Upstream)
	if err != nil || upstream.Scheme != "http" || upstream.Host == "" || upstream.User != nil || upstream.RawQuery != "" || upstream.Fragment != "" || (upstream.Path != "" && upstream.Path != "/") {
		return nil, fmt.Errorf("gate requires an internal HTTP upstream URL")
	}
	if log == nil {
		return nil, fmt.Errorf("gate requires a structured logger")
	}
	// Changing either secret or password hash invalidates all existing sessions.
	derivation := hmac.New(sha256.New, secret)
	derivation.Write([]byte("flowkit-session-v1\x00" + cfg.PasswordHash))
	g := &Gate{hash: []byte(cfg.PasswordHash), key: derivation.Sum(nil), log: log, now: time.Now, clients: make(map[string]loginCounter)}
	g.proxy = &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(upstream)
			r.SetXForwarded()
			r.Out.Header.Set("X-Forwarded-Proto", "https")
			// Credentials and the gate's session never reach Nginx or the API.
			r.Out.Header.Del("Authorization")
			r.Out.Header.Del("Cookie")
			r.Out.Header.Del("X-FlowKit-Client-IP")
		},
		ErrorLog: slog.NewLogLogger(log.Handler(), slog.LevelError),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error("Dashboard upstream unavailable", "event", "gate.upstream_failed")
			writeError(w, http.StatusServiceUnavailable, "UPSTREAM_UNAVAILABLE", "Servise şu anda ulaşılamıyor.")
		},
		ModifyResponse: func(r *http.Response) error {
			r.Header.Set("Cache-Control", "no-store")
			r.Header.Add("Vary", "Cookie")
			return nil
		},
	}
	return g, nil
}

type responseStatus struct {
	http.ResponseWriter
	status int
}

func (w *responseStatus) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *responseStatus) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}
func (w *responseStatus) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (g *Gate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	tracked := &responseStatus{ResponseWriter: w}
	defer func() {
		g.log.Info("Gate request", "event", "gate.request", "method", r.Method, "path", r.URL.Path, "status", tracked.status, "duration_ms", time.Since(start).Milliseconds())
	}()
	w = tracked
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Add("Vary", "Cookie")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	switch r.URL.Path {
	case "/auth/login":
		g.login(w, r)
		return
	case "/auth/session":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, "GET")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"enabled": true, "authenticated": g.authenticated(r)})
		return
	case "/auth/logout":
		if r.Method != http.MethodPost {
			methodNotAllowed(w, "POST")
			return
		}
		if !sameOrigin(r, true) {
			writeError(w, http.StatusForbidden, "ORIGIN_REJECTED", "Bu kaynaktan işlem kabul edilmiyor.")
			return
		}
		http.SetCookie(w, sessionCookie("", time.Unix(1, 0), -1))
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	if !g.authenticated(r) {
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		} else {
			writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Devam etmek için giriş yapın.")
		}
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions && !sameOrigin(r, false) {
		writeError(w, http.StatusForbidden, "ORIGIN_REJECTED", "Bu kaynaktan işlem kabul edilmiyor.")
		return
	}
	g.proxy.ServeHTTP(w, r)
}

func sameOrigin(r *http.Request, required bool) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return !required
	} // Cookie-authenticated CLI API calls have no Origin.
	return origin == "https://"+r.Host
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Bu yöntem desteklenmiyor.")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func sessionCookie(value string, expires time.Time, maxAge int) *http.Cookie {
	return &http.Cookie{Name: cookieName, Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: maxAge}
}

func (g *Gate) signature(payload string) string {
	m := hmac.New(sha256.New, g.key)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (g *Gate) authenticated(r *http.Request) bool {
	cookie, err := r.Cookie(cookieName)
	if err != nil || len(cookie.Value) > 160 {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return false
	}
	expires, err := strconv.ParseInt(parts[0], 10, 64)
	now := g.now().Unix()
	if err != nil || expires <= now || expires > now+int64(sessionTTL/time.Second)+1 {
		return false
	}
	nonce, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(nonce) != 24 {
		return false
	}
	return hmac.Equal([]byte(parts[2]), []byte(g.signature(parts[0]+"."+parts[1])))
}

func (g *Gate) loginAllowed(r *http.Request) bool {
	// Only the private Caddy proxy can reach this service; it overwrites this header.
	ip := net.ParseIP(r.Header.Get("X-FlowKit-Client-IP"))
	if ip == nil {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip = net.ParseIP(host)
	}
	client := ip.String()
	now := g.now()
	g.mu.Lock()
	defer g.mu.Unlock()
	for key, counter := range g.clients {
		if now.Sub(counter.start) >= loginWindow {
			delete(g.clients, key)
		}
	}
	counter, exists := g.clients[client]
	if !exists {
		if len(g.clients) >= maxClients {
			return false
		}
		counter.start = now
	}
	if counter.count >= loginLimit {
		return false
	}
	counter.count++
	g.clients[client] = counter
	return true
}

func (g *Gate) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if g.authenticated(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		loginPage(w, http.StatusOK, "")
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "GET, POST")
		return
	}
	if !sameOrigin(r, true) {
		writeError(w, http.StatusForbidden, "ORIGIN_REJECTED", "Bu kaynaktan işlem kabul edilmiyor.")
		return
	}
	if !g.loginAllowed(r) {
		w.Header().Set("Retry-After", "60")
		loginPage(w, http.StatusTooManyRequests, "Çok fazla deneme. Bir dakika sonra tekrar deneyin.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	contentType, _, typeError := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err := r.ParseForm(); err != nil || typeError != nil || contentType != "application/x-www-form-urlencoded" {
		loginPage(w, http.StatusBadRequest, "Giriş formu geçersiz.")
		return
	}
	password := r.PostForm.Get("password")
	if len(password) == 0 || len(password) > 72 || bcrypt.CompareHashAndPassword(g.hash, []byte(password)) != nil {
		g.log.Warn("Login rejected", "event", "gate.login_rejected")
		loginPage(w, http.StatusUnauthorized, "Şifre doğru değil.")
		return
	}
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		g.log.Error("Session creation failed", "event", "gate.session_failed")
		writeError(w, http.StatusInternalServerError, "SESSION_FAILED", "Giriş tamamlanamadı.")
		return
	}
	expires := g.now().Add(sessionTTL)
	payload := strconv.FormatInt(expires.Unix(), 10) + "." + base64.RawURLEncoding.EncodeToString(nonce)
	http.SetCookie(w, sessionCookie(payload+"."+g.signature(payload), expires, int(sessionTTL/time.Second)))
	g.log.Info("Login accepted", "event", "gate.login_accepted")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

var page = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>FlowKit · Giriş</title>
<style>
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;background:#f5f6f9;color:#192338;font:16px system-ui,-apple-system,sans-serif;padding:24px}.card{width:100%;max-width:420px;background:#fff;border:1px solid #e4e8f0;border-radius:20px;padding:36px;box-shadow:0 12px 40px #14244408}.brand{font-size:25px;font-weight:750;letter-spacing:-1px}.mark{display:inline-grid;place-items:center;width:38px;height:38px;border-radius:12px;background:#5069e8;color:#fff;margin-right:10px}.tag{font-size:10px;letter-spacing:2px;color:#8c94a5;margin-left:10px}h1{font-size:23px;margin:30px 0 10px}p{color:#717b90;font-size:14px;line-height:1.6;margin:0 0 24px}label{font-size:13px;font-weight:600;display:block;margin-bottom:9px}input{width:100%;border:1px solid #dce1eb;border-radius:10px;padding:13px;font:inherit;background:#fcfcfe}input:focus{outline:2px solid #5069e8;outline-offset:2px}button{margin-top:20px;width:100%;padding:13px;border:0;border-radius:10px;background:#5069e8;color:#fff;font:600 14px system-ui;cursor:pointer}button:hover{background:#4159d4}.error{background:#fff0ef;color:#a33630;border-radius:8px;padding:12px;font-size:13px;margin-bottom:20px}footer{font-size:12px;color:#929bad;text-align:center;margin-top:24px}
</style></head><body><main class="card"><div class="brand"><span class="mark">ƒ</span>FlowKit<span class="tag">STUDIO</span></div><h1>Çalışma alanına giriş</h1><p>Devam etmek için FlowKit erişim şifresini girin.</p>{{if .}}<div class="error" role="alert">{{.}}</div>{{end}}<form method="post" action="/auth/login"><label for="password">Şifre</label><input id="password" name="password" type="password" autocomplete="current-password" required autofocus maxlength="72"><button type="submit">Giriş yap</button></form><footer>FlowKit · Workflow yönetim paneli</footer></main></body></html>`))

func loginPage(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	w.WriteHeader(status)
	_ = page.Execute(w, message)
}
