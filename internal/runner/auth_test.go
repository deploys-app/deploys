package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/deploys-app/api"

	"github.com/deploys-app/deploys/internal/auth"
)

func tempOut(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func readOut(t *testing.T, f *os.File) string {
	t.Helper()
	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// meGetServer returns an httptest server answering me.get with the arpc envelope.
func meGetServer(t *testing.T, email string, ok bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"result":{"email":"` + email + `","kyc":false}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestIsAuthCommand(t *testing.T) {
	for _, n := range []string{"login", "logout", "auth"} {
		if !IsAuthCommand(n) {
			t.Errorf("IsAuthCommand(%q) = false; want true", n)
		}
	}
	for _, n := range []string{"me", "deployment", "version", "check-update", ""} {
		if IsAuthCommand(n) {
			t.Errorf("IsAuthCommand(%q) = true; want false", n)
		}
	}
}

func TestLoginGlue(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	me := meGetServer(t, "alice@example.com", true)
	t.Setenv("DEPLOYS_ENDPOINT", me.URL)
	t.Setenv("DEPLOYS_AUTH_ENDPOINT", "https://auth.deploys.app")

	old := doLogin
	doLogin = func(ctx context.Context, authBase, apiEndpoint string, opts auth.LoginOptions) (auth.Result, error) {
		return auth.Result{Token: "deploys-api.glue", ExpiresIn: 7 * 24 * time.Hour}, nil
	}
	defer func() { doLogin = old }()

	rn := Runner{Output: tempOut(t)}
	if err := rn.login(); err != nil {
		t.Fatalf("login: %v", err)
	}
	out := readOut(t, rn.Output)
	if !strings.Contains(out, "Logged in as alice@example.com") {
		t.Errorf("login output missing identity:\n%s", out)
	}

	c, err := auth.Load()
	if err != nil {
		t.Fatal(err)
	}
	key := auth.AccountKey(me.URL, "alice@example.com")
	a, ok := c.Find(key)
	if !ok {
		t.Fatalf("account not persisted; store=%+v", c.Accounts)
	}
	if a.Token != "deploys-api.glue" {
		t.Errorf("token = %q", a.Token)
	}
	if a.AuthEndpoint != "https://auth.deploys.app" {
		t.Errorf("authEndpoint = %q", a.AuthEndpoint)
	}
	if k, ok := c.ActiveKey(me.URL); !ok || k != key {
		t.Errorf("active not set to new account: %q ok=%v", k, ok)
	}
}

func TestLoginMeGetFailurePersists(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	me := meGetServer(t, "", false) // 500 -> me.get fails
	t.Setenv("DEPLOYS_ENDPOINT", me.URL)

	old := doLogin
	doLogin = func(ctx context.Context, authBase, apiEndpoint string, opts auth.LoginOptions) (auth.Result, error) {
		return auth.Result{Token: "deploys-api.noident", ExpiresIn: 7 * 24 * time.Hour}, nil
	}
	defer func() { doLogin = old }()

	rn := Runner{Output: tempOut(t)}
	if err := rn.login(); err != nil {
		t.Fatalf("login should still succeed when me.get fails: %v", err)
	}
	c, _ := auth.Load()
	if len(c.Accounts) != 1 {
		t.Fatalf("token not persisted under placeholder; accounts=%d", len(c.Accounts))
	}
	if c.Accounts[0].Token != "deploys-api.noident" {
		t.Errorf("placeholder account token = %q", c.Accounts[0].Token)
	}
	if k, ok := c.ActiveKey(me.URL); !ok || k != c.Accounts[0].Key {
		t.Errorf("placeholder account not active")
	}
}

func TestAuthListAndStatus(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	// Ensure no env credential shadows the file source for status.
	t.Setenv("DEPLOYS_TOKEN", "")
	t.Setenv("DEPLOYS_AUTH_USER", "")
	t.Setenv("DEPLOYS_AUTH_PASS", "")
	ep := "https://api.deploys.app/"
	t.Setenv("DEPLOYS_ENDPOINT", ep)

	if err := auth.Mutate(func(cc *auth.Credentials) error {
		cc.Upsert(auth.Account{Key: auth.AccountKey(ep, "alice@x"), Endpoint: ep, Email: "alice@x", Token: "t1", ExpiresAt: time.Now().Add(48 * time.Hour)})
		cc.Upsert(auth.Account{Key: auth.AccountKey(ep, "bob@x"), Endpoint: ep, Email: "bob@x", Token: "t2", ExpiresAt: time.Now().Add(48 * time.Hour)})
		cc.SetActive(ep, auth.AccountKey(ep, "alice@x"))
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// list
	rn := Runner{Output: tempOut(t)}
	if err := rn.authList(); err != nil {
		t.Fatalf("authList: %v", err)
	}
	out := readOut(t, rn.Output)
	for _, want := range []string{"ACTIVE", "ENDPOINT", "ACCOUNT", "EXPIRES", "alice@x", "bob@x"} {
		if !strings.Contains(out, want) {
			t.Errorf("auth list missing %q:\n%s", want, out)
		}
	}

	// status -> active account from the file
	rn2 := Runner{Output: tempOut(t)}
	if err := rn2.authStatus(); err != nil {
		t.Fatalf("authStatus: %v", err)
	}
	st := readOut(t, rn2.Output)
	if !strings.Contains(st, "credentials file") || !strings.Contains(st, "alice@x") {
		t.Errorf("auth status missing active account:\n%s", st)
	}
}

func TestAuthStatusEnvToken(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	t.Setenv("DEPLOYS_TOKEN", "deploys-api.env")
	rn := Runner{Output: tempOut(t)}
	if err := rn.authStatus(); err != nil {
		t.Fatalf("authStatus: %v", err)
	}
	out := readOut(t, rn.Output)
	if !strings.Contains(out, "DEPLOYS_TOKEN") {
		t.Errorf("status should report env source:\n%s", out)
	}
}

func TestAuthSwitch(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	ep := "https://api.deploys.app/"
	t.Setenv("DEPLOYS_ENDPOINT", ep)
	if err := auth.Mutate(func(cc *auth.Credentials) error {
		cc.Upsert(auth.Account{Key: auth.AccountKey(ep, "alice@x"), Endpoint: ep, Email: "alice@x", Token: "t1", ExpiresAt: time.Now().Add(time.Hour)})
		cc.Upsert(auth.Account{Key: auth.AccountKey(ep, "bob@x"), Endpoint: ep, Email: "bob@x", Token: "t2", ExpiresAt: time.Now().Add(time.Hour)})
		cc.SetActive(ep, auth.AccountKey(ep, "alice@x"))
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	rn := Runner{Output: tempOut(t), Account: "bob@x"}
	if err := rn.authSwitch(); err != nil {
		t.Fatalf("authSwitch: %v", err)
	}
	c, _ := auth.Load()
	if k, _ := c.ActiveKey(ep); k != auth.AccountKey(ep, "bob@x") {
		t.Errorf("active not switched to bob: %q", k)
	}
}

func TestAuthTokenPipeNoNewline(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	t.Setenv("DEPLOYS_TOKEN", "")
	ep := "https://api.deploys.app/"
	t.Setenv("DEPLOYS_ENDPOINT", ep)
	if err := auth.Mutate(func(cc *auth.Credentials) error {
		cc.Upsert(auth.Account{Key: auth.AccountKey(ep, "alice@x"), Endpoint: ep, Email: "alice@x", Token: "deploys-api.tok", ExpiresAt: time.Now().Add(time.Hour)})
		cc.SetActive(ep, auth.AccountKey(ep, "alice@x"))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Output is a regular file (not a TTY) -> prints without a trailing newline.
	rn := Runner{Output: tempOut(t)}
	if err := rn.authToken(); err != nil {
		t.Fatalf("authToken: %v", err)
	}
	if got := readOut(t, rn.Output); got != "deploys-api.tok" {
		t.Errorf("token output = %q; want exact token, no newline", got)
	}
}

func TestLogoutRevokesAndRemoves(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	ep := "https://api.deploys.app/"
	t.Setenv("DEPLOYS_ENDPOINT", ep)

	var revoked bool
	revokeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/revoke" {
			revoked = true
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer revokeSrv.Close()

	if err := auth.Mutate(func(cc *auth.Credentials) error {
		cc.Upsert(auth.Account{Key: auth.AccountKey(ep, "alice@x"), Endpoint: ep, AuthEndpoint: revokeSrv.URL, Email: "alice@x", Token: "deploys-api.tok", ExpiresAt: time.Now().Add(time.Hour)})
		cc.SetActive(ep, auth.AccountKey(ep, "alice@x"))
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	rn := Runner{Output: tempOut(t)}
	if err := rn.logout(); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if !revoked {
		t.Error("logout did not call /revoke")
	}
	c, _ := auth.Load()
	if len(c.Accounts) != 0 {
		t.Errorf("account not removed after logout: %+v", c.Accounts)
	}
}

func TestLogoutKeepsEntryOnRevokeFailure(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	ep := "https://api.deploys.app/"
	t.Setenv("DEPLOYS_ENDPOINT", ep)

	// An unreachable auth base makes revoke fail; the entry must be kept.
	if err := auth.Mutate(func(cc *auth.Credentials) error {
		cc.Upsert(auth.Account{Key: auth.AccountKey(ep, "alice@x"), Endpoint: ep, AuthEndpoint: "https://127.0.0.1:1", Email: "alice@x", Token: "deploys-api.tok", ExpiresAt: time.Now().Add(time.Hour)})
		cc.SetActive(ep, auth.AccountKey(ep, "alice@x"))
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	rn := Runner{Output: tempOut(t)}
	if err := rn.logout(); err == nil {
		t.Fatal("logout should fail loudly when revoke fails")
	}
	c, _ := auth.Load()
	if len(c.Accounts) != 1 {
		t.Errorf("account should be kept when revoke fails; got %d", len(c.Accounts))
	}
}

func TestLogoutNothingToDo(t *testing.T) {
	t.Setenv("DEPLOYS_CONFIG_DIR", t.TempDir())
	t.Setenv("DEPLOYS_ENDPOINT", "https://api.deploys.app/")
	rn := Runner{Output: tempOut(t)}
	if err := rn.logout(); err != nil {
		t.Fatalf("logout with no accounts should be a no-op success: %v", err)
	}
	if out := readOut(t, rn.Output); !strings.Contains(out, "nothing to log out") {
		t.Errorf("expected 'nothing to log out', got %q", out)
	}
}

// guard: api.ErrForbidden must not be confused with the unauthorized sentinel by
// callers; this documents the distinction the exit-code mapper relies on.
func TestUnauthorizedSentinelDistinct(t *testing.T) {
	if api.ErrUnauthorized == api.ErrForbidden {
		t.Fatal("ErrUnauthorized and ErrForbidden must be distinct sentinels")
	}
}
