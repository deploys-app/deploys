package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAccountKey(t *testing.T) {
	// Endpoint normalization collides the trailing-slash and bare forms.
	a := AccountKey("https://api.deploys.app", "alice@example.com")
	b := AccountKey("https://api.deploys.app/", "alice@example.com")
	if a != b {
		t.Fatalf("keys differ across trailing slash: %q vs %q", a, b)
	}
	// Separator is a NUL byte, not a space.
	if !strings.Contains(a, "\x00") {
		t.Errorf("key missing NUL separator: %q", a)
	}
	// Empty endpoint resolves to the default.
	if got := AccountKey("", "x@y"); !strings.HasPrefix(got, defaultEndpoint) {
		t.Errorf("empty endpoint key = %q; want default-prefixed", got)
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", defaultEndpoint},
		{"https://api.deploys.app", "https://api.deploys.app/"},
		{"https://api.deploys.app/", "https://api.deploys.app/"},
		{"https://api.staging:8443", "https://api.staging:8443/"},
	}
	for _, c := range cases {
		if got := normalizeEndpoint(c.in); got != c.want {
			t.Errorf("normalizeEndpoint(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestGeneratePKCE(t *testing.T) {
	v, c := generatePKCE()
	if len(v) < 43 || len(v) > 128 {
		t.Errorf("verifier length %d out of [43,128]", len(v))
	}
	sum := sha256.Sum256([]byte(v))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if c != want {
		t.Errorf("challenge = %q; want %q", c, want)
	}
	if strings.ContainsAny(v, "=+/") || strings.ContainsAny(c, "=+/") {
		t.Errorf("PKCE values must be unpadded base64url: v=%q c=%q", v, c)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DEPLOYS_CONFIG_DIR", dir)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if len(c.Accounts) != 0 {
		t.Fatalf("fresh store should be empty, got %d", len(c.Accounts))
	}

	ep := "https://api.deploys.app/"
	acc := Account{
		Key:          AccountKey(ep, "alice@example.com"),
		Endpoint:     ep,
		AuthEndpoint: "https://auth.deploys.app",
		Email:        "alice@example.com",
		Token:        "deploys-api.abc",
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		IssuedAt:     time.Now(),
	}
	if err := Mutate(func(cc *Credentials) error {
		cc.Upsert(acc)
		cc.SetActive(ep, acc.Key)
		return nil
	}); err != nil {
		t.Fatalf("Mutate: %v", err)
	}

	// File mode is 0600 and no temp file is left behind.
	path := filepath.Join(dir, "credentials.json")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat credentials: %v", err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Errorf("credentials mode = %v; want 0600", fi.Mode().Perm())
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}

	// Reload and assert content + active selection.
	c2, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, ok := c2.Find(acc.Key)
	if !ok || got.Token != "deploys-api.abc" {
		t.Fatalf("account not round-tripped: %+v ok=%v", got, ok)
	}
	if k, ok := c2.ActiveKey(ep); !ok || k != acc.Key {
		t.Errorf("active key = %q ok=%v; want %q", k, ok, acc.Key)
	}

	// Upsert replaces by key (no duplicate).
	acc.Token = "deploys-api.def"
	if err := Mutate(func(cc *Credentials) error { cc.Upsert(acc); return nil }); err != nil {
		t.Fatalf("Mutate replace: %v", err)
	}
	c3, _ := Load()
	if len(c3.Accounts) != 1 {
		t.Errorf("expected 1 account after replace, got %d", len(c3.Accounts))
	}
	if a, _ := c3.Find(acc.Key); a.Token != "deploys-api.def" {
		t.Errorf("replace did not update token: %q", a.Token)
	}
}

func TestRemoveReassignsActive(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DEPLOYS_CONFIG_DIR", dir)

	ep := "https://api.deploys.app/"
	mk := func(email string) Account {
		return Account{Key: AccountKey(ep, email), Endpoint: ep, Email: email, Token: "t-" + email, ExpiresAt: time.Now().Add(time.Hour)}
	}
	alice, bob := mk("alice@x"), mk("bob@x")
	if err := Mutate(func(cc *Credentials) error {
		cc.Upsert(alice)
		cc.Upsert(bob)
		cc.SetActive(ep, alice.Key)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Removing the active account repoints active to the remaining one.
	if err := Mutate(func(cc *Credentials) error {
		cc.Remove(alice.Key)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	c, _ := Load()
	if k, ok := c.ActiveKey(ep); !ok || k != bob.Key {
		t.Errorf("active after removing alice = %q ok=%v; want bob", k, ok)
	}
}

func TestLoadCorruptAndVersion(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DEPLOYS_CONFIG_DIR", dir)
	path := filepath.Join(dir, "credentials.json")

	// Missing file -> empty, not error.
	if _, err := Load(); err != nil {
		t.Fatalf("missing file should be empty creds, got %v", err)
	}

	// Malformed -> error (never silently empty).
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Error("malformed file should error")
	}

	// Higher version -> error.
	if err := os.WriteFile(path, []byte(`{"version":999,"accounts":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("higher version should error mentioning version, got %v", err)
	}
}

func TestClientIDCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DEPLOYS_CONFIG_DIR", dir)

	if _, ok, err := LoadClientID("https://auth.deploys.app"); err != nil || ok {
		t.Fatalf("empty cache: ok=%v err=%v", ok, err)
	}
	if err := SaveClientID("https://auth.deploys.app", "cid-123"); err != nil {
		t.Fatalf("SaveClientID: %v", err)
	}
	id, ok, err := LoadClientID("https://auth.deploys.app")
	if err != nil || !ok || id != "cid-123" {
		t.Fatalf("LoadClientID = %q ok=%v err=%v", id, ok, err)
	}
	// A different auth base is independent.
	if _, ok, _ := LoadClientID("https://auth.other"); ok {
		t.Error("client id should be keyed per auth base")
	}
}

func TestMutateTightensLoosePerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix perms not enforced on windows")
	}
	dir := t.TempDir()
	t.Setenv("DEPLOYS_CONFIG_DIR", dir)
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"accounts":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// A read should tighten the loose-but-owned file to 0600.
	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	fi, _ := os.Stat(path)
	if fi.Mode().Perm()&0o077 != 0 {
		t.Errorf("perms not tightened: %v", fi.Mode().Perm())
	}
}

func TestLoadRefusesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	dir := t.TempDir()
	t.Setenv("DEPLOYS_CONFIG_DIR", dir)
	target := filepath.Join(dir, "real.json")
	if err := os.WriteFile(target, []byte(`{"version":1,"accounts":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "credentials.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected symlink refusal, got %v", err)
	}
}

func TestExpired(t *testing.T) {
	if (Account{ExpiresAt: time.Now().Add(time.Hour)}).Expired() {
		t.Error("future expiry should not be expired")
	}
	if !(Account{ExpiresAt: time.Now().Add(-time.Hour)}).Expired() {
		t.Error("past expiry should be expired")
	}
	if (Account{}).Expired() {
		t.Error("zero expiry should be treated as non-expiring")
	}
}

func TestResolveExplicitVsImplicit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DEPLOYS_CONFIG_DIR", dir)

	ep := "https://api.deploys.app/"
	staging := "https://api.staging:8443/"
	now := time.Now()
	if err := Mutate(func(cc *Credentials) error {
		cc.Upsert(Account{Key: AccountKey(ep, "alice@x"), Endpoint: ep, Email: "alice@x", Token: "tok-alice", ExpiresAt: now.Add(time.Hour)})
		cc.Upsert(Account{Key: AccountKey(ep, "bob@x"), Endpoint: ep, Email: "bob@x", Token: "tok-bob", ExpiresAt: now.Add(time.Hour)})
		cc.Upsert(Account{Key: AccountKey(ep, "old@x"), Endpoint: ep, Email: "old@x", Token: "tok-old", ExpiresAt: now.Add(-time.Hour)})
		cc.Upsert(Account{Key: AccountKey(staging, "alice@x"), Endpoint: staging, Email: "alice@x", Token: "tok-staging", ExpiresAt: now.Add(time.Hour)})
		cc.SetActive(ep, AccountKey(ep, "alice@x"))
		cc.SetActive(staging, AccountKey(staging, "alice@x"))
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Active default on prod.
	if a, expired, err := Resolve("", ep, false); err != nil || expired || a == nil || a.Token != "tok-alice" {
		t.Errorf("active prod = %+v expired=%v err=%v", a, expired, err)
	}
	// Per-endpoint active map keeps staging independent.
	if a, _, err := Resolve("", staging, false); err != nil || a == nil || a.Token != "tok-staging" {
		t.Errorf("active staging = %+v err=%v", a, err)
	}
	// Explicit selector picks a non-active account.
	if a, _, err := Resolve("bob@x", ep, true); err != nil || a == nil || a.Token != "tok-bob" {
		t.Errorf("explicit bob = %+v err=%v", a, err)
	}
	// Implicit miss (selector for nonexistent, but implicit) -> nothing, no error.
	if a, expired, err := Resolve("ghost@x", "https://api.unknown/", false); err != nil || a != nil || expired {
		t.Errorf("implicit miss should be (nil,false,nil); got %+v %v %v", a, expired, err)
	}
	// Explicit miss -> hard AuthRequiredError.
	_, _, err := Resolve("ghost@x", ep, true)
	var are *AuthRequiredError
	if !errors.As(err, &are) {
		t.Errorf("explicit miss should be AuthRequiredError, got %v", err)
	}
	// Implicit expired active -> account returned with expired=true, no token use.
	if err := Mutate(func(cc *Credentials) error {
		cc.SetActive(ep, AccountKey(ep, "old@x"))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	a, expired, err := Resolve("", ep, false)
	if err != nil || !expired || a == nil {
		t.Errorf("implicit expired = %+v expired=%v err=%v; want expired account", a, expired, err)
	}
	// Explicit expired -> hard error.
	_, _, err = Resolve("old@x", ep, true)
	if !errors.As(err, &are) {
		t.Errorf("explicit expired should be AuthRequiredError, got %v", err)
	}
}
