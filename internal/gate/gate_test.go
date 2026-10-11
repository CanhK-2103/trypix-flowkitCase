package gate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const testPassword = "independent-site-password"

func setup(t *testing.T) (*Gate, Config, *bytes.Buffer) {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("X-FlowKit-Client-IP") != "" {
			t.Error("gate leaked credentials to upstream")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"path": r.URL.Path, "method": r.Method})
	}))
	t.Cleanup(upstream.Close)
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{PasswordHash: string(hash), SessionKey: strings.Repeat("ab", 32), Upstream: upstream.URL}
	logs := &bytes.Buffer{}
	log := slog.New(slog.NewJSONHandler(logs, nil)).With("run_id", nil, "node_id", nil, "attempt", nil)
	g, err := New(cfg, log)
	if err != nil {
		t.Fatal(err)
	}
	return g, cfg, logs
}

func call(g *Gate, method, path, body, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "https://flowkit.test"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	return w
}

func signIn(t *testing.T, g *Gate) *http.Cookie {
	t.Helper()
	w := call(g, "POST", "/auth/login", "password="+url.QueryEscape(testPassword), "https://flowkit.test", nil)
	if w.Code != 303 || w.Header().Get("Location") != "/" {
		t.Fatalf("login failed: %d", w.Code)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	return cookies[0]
}

func TestProtectedPathsAndPasswordForm(t *testing.T) {
	g, _, _ := setup(t)
	root := call(g, "GET", "/", "", "", nil)
	if root.Code != 303 || root.Header().Get("Location") != "/auth/login" {
		t.Fatal("root must redirect to login")
	}
	for _, path := range []string{"/models", "/workflows", "/runs", "/health/ready", "/artifacts/x", "/assets/x.js"} {
		w := call(g, "GET", path, "", "", nil)
		if w.Code != 401 || !strings.Contains(w.Body.String(), "AUTH_REQUIRED") {
			t.Fatalf("unprotected %s", path)
		}
		if w.Header().Get("WWW-Authenticate") != "" {
			t.Fatal("must not show a username dialog")
		}
	}
	form := call(g, "GET", "/auth/login", "", "", nil)
	if form.Code != 200 || strings.Contains(form.Body.String(), "username") || strings.Count(form.Body.String(), "<input") != 1 {
		t.Fatal("expected password-only form")
	}
	if form.Header().Get("Cache-Control") != "no-store" || !strings.Contains(form.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatal("missing login protection headers")
	}
	wrong := call(g, "POST", "/auth/login", "password=wrong", "https://flowkit.test", nil)
	if wrong.Code != 401 || len(wrong.Result().Cookies()) != 0 || strings.Contains(wrong.Body.String(), "value=\"wrong\"") {
		t.Fatal("invalid password accepted or reflected")
	}
}

func TestSessionProxyExpiryAndRotation(t *testing.T) {
	g, cfg, _ := setup(t)
	cookie := signIn(t, g)
	if cookie.Name != cookieName || !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Domain != "" || cookie.MaxAge != 43200 {
		t.Fatal("unsafe session cookie")
	}
	r := httptest.NewRequest("POST", "https://flowkit.test/workflows", strings.NewReader("{}"))
	r.AddCookie(cookie)
	r.Header.Set("Authorization", "secret-must-not-reach-upstream")
	r.Header.Set("X-FlowKit-Client-IP", "192.0.2.1")
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"path":"/workflows"`) || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("authenticated proxy failed")
	}
	restarted, err := New(cfg, g.log)
	if err != nil {
		t.Fatal(err)
	}
	if call(restarted, "GET", "/models", "", "", cookie).Code != 200 {
		t.Fatal("restart lost persistent session")
	}
	bad := *cookie
	bad.Value = "1" + cookie.Value
	if call(g, "GET", "/models", "", "", &bad).Code != 401 {
		t.Fatal("tampered session accepted")
	}
	bad.Value = strings.Repeat("x", 200)
	if call(g, "GET", "/models", "", "", &bad).Code != 401 {
		t.Fatal("oversized session accepted")
	}
	cfg.SessionKey = strings.Repeat("cd", 32)
	rotated, _ := New(cfg, g.log)
	if call(rotated, "GET", "/models", "", "", cookie).Code != 401 {
		t.Fatal("key rotation failed to invalidate session")
	}
	cfg.SessionKey = strings.Repeat("ab", 32)
	hash, _ := bcrypt.GenerateFromPassword([]byte("a-new-site-password"), bcrypt.MinCost)
	cfg.PasswordHash = string(hash)
	rotated, _ = New(cfg, g.log)
	if call(rotated, "GET", "/models", "", "", cookie).Code != 401 {
		t.Fatal("password rotation failed to invalidate session")
	}
	clock := g.now().Add(sessionTTL + time.Second)
	g.now = func() time.Time { return clock }
	if call(g, "GET", "/models", "", "", cookie).Code != 401 {
		t.Fatal("expired session accepted")
	}
}

func TestCrossOriginAndLogout(t *testing.T) {
	g, _, _ := setup(t)
	for _, origin := range []string{"", "https://attacker.test", "null", "http://flowkit.test", "https://flowkit.test.attacker.test"} {
		if call(g, "POST", "/auth/login", "password="+testPassword, origin, nil).Code != 403 {
			t.Fatalf("unsafe login origin %q", origin)
		}
	}
	cookie := signIn(t, g)
	for _, path := range []string{"/auth/logout", "/workflows"} {
		if call(g, "POST", path, "", "https://attacker.test", cookie).Code != 403 {
			t.Fatalf("unsafe mutation %s", path)
		}
	}
	r := httptest.NewRequest("POST", "https://flowkit.test/workflows", nil)
	r.AddCookie(cookie)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-site fetch accepted")
	}
	if call(g, "GET", "/auth/logout", "", "", cookie).Code != 405 {
		t.Fatal("GET must not log out")
	}
	logout := call(g, "POST", "/auth/logout", "", "https://flowkit.test", cookie)
	if logout.Code != 303 || len(logout.Result().Cookies()) != 1 || logout.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("logout did not expire cookie")
	}
	if call(g, "GET", "/models", "", "", logout.Result().Cookies()[0]).Code != 401 {
		t.Fatal("logout cookie remained authenticated")
	}
}

func TestRateLimitAndBoundedStorage(t *testing.T) {
	g, _, _ := setup(t)
	for i := 0; i < loginLimit; i++ {
		if call(g, "POST", "/auth/login", "password=wrong", "https://flowkit.test", nil).Code != 401 {
			t.Fatal("premature throttle")
		}
	}
	limited := call(g, "POST", "/auth/login", "password="+testPassword, "https://flowkit.test", nil)
	if limited.Code != 429 || limited.Header().Get("Retry-After") == "" {
		t.Fatal("password attempts not bounded")
	}
	clock := g.now().Add(loginWindow + time.Second)
	g.now = func() time.Time { return clock }
	signIn(t, g)
	g.clients = make(map[string]loginCounter)
	for i := 0; i < maxClients; i++ {
		g.clients[fmt.Sprint(i)] = loginCounter{start: clock}
	}
	if call(g, "POST", "/auth/login", "password="+testPassword, "https://flowkit.test", nil).Code != 429 || len(g.clients) != maxClients {
		t.Fatal("limiter memory limit failed")
	}
	clock = clock.Add(loginWindow + time.Second)
	signIn(t, g)
}

func TestConcurrentLimiter(t *testing.T) {
	g, _, _ := setup(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 20; i++ {
		wg.Go(func() {
			r := httptest.NewRequest("POST", "https://flowkit.test/auth/login", nil)
			if g.loginAllowed(r) {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if allowed != loginLimit {
		t.Fatalf("concurrent attempts bypassed limit: %d", allowed)
	}
}

func TestFailClosedConfigurationAndSanitizedLogs(t *testing.T) {
	g, cfg, logs := setup(t)
	for _, invalid := range []Config{{}, {PasswordHash: cfg.PasswordHash, SessionKey: "short", Upstream: cfg.Upstream}, {PasswordHash: cfg.PasswordHash, SessionKey: cfg.SessionKey, Upstream: "http://user:password@localhost"}} {
		if _, err := New(invalid, g.log); err == nil {
			t.Fatal("invalid configuration allowed startup")
		}
	}
	if _, err := New(cfg, nil); err == nil {
		t.Fatal("missing logger accepted")
	}
	cookie := signIn(t, g)
	call(g, "POST", "/auth/login", "password=do-not-log-this", "https://flowkit.test", nil)
	call(g, "GET", "/models?secret=do-not-log-this", "", "", cookie)
	for _, secret := range []string{testPassword, cfg.PasswordHash, cfg.SessionKey, cookie.Value, "do-not-log-this"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatal("sensitive value in logs")
		}
	}
	decoder := json.NewDecoder(logs)
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"run_id", "node_id", "attempt"} {
			if value, exists := entry[key]; !exists || value != nil {
				t.Fatalf("missing null log context %s", key)
			}
		}
	}
}

func TestInvalidBodyAndUnavailableUpstream(t *testing.T) {
	g, _, _ := setup(t)
	if call(g, "POST", "/auth/login", "password="+strings.Repeat("a", 1025), "https://flowkit.test", nil).Code != 400 {
		t.Fatal("oversized body accepted")
	}
	cookie := signIn(t, g)
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	upstream.Close()
	u, _ := url.Parse(upstream.URL)
	g.proxy.Rewrite = func(r *httputil.ProxyRequest) { r.SetURL(u) }
	if call(g, "GET", "/models", "", "", cookie).Code != 503 {
		t.Fatal("unavailable upstream not handled")
	}
}
