package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

// fakeAuthServer is a minimal OAuth 2.1 + PKCE authorization server good enough
// to drive the CLI flow end to end: discovery, DCR, authorize (which 302s to the
// loopback redirect), and token (which verifies PKCE S256).
func fakeAuthServer(t *testing.T) (*httptest.Server, *authProbe) {
	t.Helper()
	probe := &authProbe{challenges: map[string]string{}}
	mux := http.NewServeMux()

	var base string // set after the server starts
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                           base,
			"authorization_endpoint":           base + "/authorize",
			"token_endpoint":                   base + "/token",
			"registration_endpoint":            base + "/register",
			"revocation_endpoint":              base + "/revoke",
			"code_challenge_methods_supported": []string{"S256"},
			"grant_types_supported":            []string{"authorization_code"},
		})
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		probe.registered++
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"client_id": "test-client"})
	})
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		probe.mu.Lock()
		probe.method = q.Get("code_challenge_method")
		probe.gotClientID = q.Get("client_id")
		code := fmt.Sprintf("code-%d", probe.registered+1)
		probe.challenges[code] = q.Get("code_challenge")
		probe.mu.Unlock()
		redirect := q.Get("redirect_uri")
		http.Redirect(w, r, redirect+"?code="+code+"&state="+url.QueryEscape(q.Get("state")), http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		probe.mu.Lock()
		reject := probe.rejectClientID
		probe.mu.Unlock()
		if reject != "" && r.Form.Get("client_id") == reject {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_client"})
			return
		}
		code := r.Form.Get("code")
		verifier := r.Form.Get("code_verifier")
		probe.mu.Lock()
		want := probe.challenges[code]
		probe.mu.Unlock()
		sum := sha256.Sum256([]byte(verifier))
		if verifier == "" || base64.RawURLEncoding.EncodeToString(sum[:]) != want {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}
		probe.exchanged = true
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "deploys-api.test",
			"token_type":   "Bearer",
			"expires_in":   604800,
		})
	})

	srv := httptest.NewServer(mux)
	base = srv.URL
	return srv, probe
}

type authProbe struct {
	mu             sync.Mutex
	challenges     map[string]string
	method         string
	gotClientID    string
	registered     int
	exchanged      bool
	rejectClientID string // /token returns invalid_client for this client_id
}

func TestLogin(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	srv, probe := fakeAuthServer(t)
	defer srv.Close()

	// The injected "browser" just GETs the authorize URL; the default http client
	// follows the 302 to the loopback /callback, delivering the code.
	opener := func(rawURL string) error {
		resp, err := http.Get(rawURL)
		if resp != nil {
			resp.Body.Close()
		}
		return err
	}

	res, err := Login(context.Background(), srv.URL, srv.URL, LoginOptions{
		OpenBrowser: opener,
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if res.Token != "deploys-api.test" {
		t.Errorf("token = %q; want deploys-api.test", res.Token)
	}
	if res.ExpiresIn != 7*24*time.Hour {
		t.Errorf("expiresIn = %v; want 7d", res.ExpiresIn)
	}
	if probe.method != "S256" {
		t.Errorf("authorize code_challenge_method = %q; want S256", probe.method)
	}
	if !probe.exchanged {
		t.Error("token endpoint did not verify/exchange the code")
	}
	// The client_id was registered once and cached.
	if probe.registered != 1 {
		t.Errorf("registered %d times; want 1", probe.registered)
	}
	if id, ok, _ := LoadClientID(srv.URL); !ok || id != "test-client" {
		t.Errorf("client id not cached: id=%q ok=%v", id, ok)
	}

	// A second login reuses the cached client_id (no re-registration).
	if _, err := Login(context.Background(), srv.URL, srv.URL, LoginOptions{OpenBrowser: opener, Timeout: 5 * time.Second}); err != nil {
		t.Fatalf("second Login: %v", err)
	}
	if probe.registered != 1 {
		t.Errorf("re-registered (count=%d); cached client_id should be reused", probe.registered)
	}
}

func TestBakedInClientID(t *testing.T) {
	if got := bakedInClientID("https://auth.deploys.app"); got != "deploys-cli" {
		t.Errorf("baked-in for default = %q; want deploys-cli", got)
	}
	for _, b := range []string{"https://auth.staging", "http://127.0.0.1:9000", ""} {
		if got := bakedInClientID(b); got != "" {
			t.Errorf("baked-in for %q = %q; want empty (DCR)", b, got)
		}
	}
}

// A stale cached/baked-in client_id rejected at /token triggers a single DCR
// re-registration and retry, so login self-heals against an un-seeded server.
func TestLoginFallsBackToDCROnInvalidClient(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	srv, probe := fakeAuthServer(t)
	defer srv.Close()

	// Pre-cache a client_id the server will reject, so ensureClientID returns it
	// as reusable and Login must fall back to DCR.
	if err := SaveClientID(srv.URL, "stale-id"); err != nil {
		t.Fatal(err)
	}
	probe.mu.Lock()
	probe.rejectClientID = "stale-id"
	probe.mu.Unlock()

	opener := func(rawURL string) error {
		resp, err := http.Get(rawURL)
		if resp != nil {
			resp.Body.Close()
		}
		return err
	}
	res, err := Login(context.Background(), srv.URL, srv.URL, LoginOptions{OpenBrowser: opener, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Login should self-heal via DCR: %v", err)
	}
	if res.Token != "deploys-api.test" {
		t.Errorf("token = %q", res.Token)
	}
	if probe.registered != 1 {
		t.Errorf("expected exactly one DCR fallback registration, got %d", probe.registered)
	}
	// The freshly registered id replaced the stale one in the cache.
	if id, ok, _ := LoadClientID(srv.URL); !ok || id == "stale-id" {
		t.Errorf("cache not updated after fallback: id=%q ok=%v", id, ok)
	}
}

func TestLoginStateMismatchFails(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	srv, _ := fakeAuthServer(t)
	defer srv.Close()

	// A malicious/garbled callback that carries the wrong state must be rejected.
	opener := func(rawURL string) error {
		u, _ := url.Parse(rawURL)
		redirect := u.Query().Get("redirect_uri")
		resp, err := http.Get(redirect + "?code=evil&state=wrong-state")
		if resp != nil {
			resp.Body.Close()
		}
		return err
	}
	_, err := Login(context.Background(), srv.URL, srv.URL, LoginOptions{OpenBrowser: opener, Timeout: 3 * time.Second})
	if err == nil {
		t.Fatal("expected login to fail on state mismatch")
	}
}

func TestValidateBase(t *testing.T) {
	ok := []string{"https://auth.deploys.app", "http://127.0.0.1:9000", "http://localhost:8080"}
	bad := []string{"http://auth.deploys.app", "ftp://x", "://nope", "http://evil.example"}
	for _, s := range ok {
		if err := validateBase(s); err != nil {
			t.Errorf("validateBase(%q) = %v; want ok", s, err)
		}
	}
	for _, s := range bad {
		if err := validateBase(s); err == nil {
			t.Errorf("validateBase(%q) = nil; want error", s)
		}
	}
}

func TestDiscoverMetadataRejectsCrossOrigin(t *testing.T) {
	// Endpoints that are not same-origin as the issuer must be rejected so a
	// metadata doc cannot repoint the token endpoint at an exfiltration host.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 "http://127.0.0.1:1", // wrong issuer
			"authorization_endpoint": "https://evil.example/authorize",
			"token_endpoint":         "https://evil.example/token",
		})
	}))
	defer srv.Close()
	if _, err := discoverMetadata(context.Background(), srv.URL); err == nil {
		t.Error("expected discovery to reject mismatched issuer/cross-origin endpoints")
	}
}

func TestRevoke(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/revoke" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct {
			Token string `json:"token"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		got = body.Token
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	if err := Revoke(context.Background(), srv.URL, "deploys-api.tok"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if got != "deploys-api.tok" {
		t.Errorf("revoke received token %q; want deploys-api.tok", got)
	}
}
