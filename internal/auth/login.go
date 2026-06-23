package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// loopbackRedirect is the registered redirect URI. It is port-less: the auth
// server's loopback rule matches any port for a 127.0.0.1 host, so a single
// registration is reused across logins on arbitrary ephemeral ports. The runtime
// redirect_uri carries the real port; both share this host and path.
const loopbackRedirect = "http://127.0.0.1/callback"

// loopbackHost is the only address the callback listener ever binds. 127.0.0.1
// (never 0.0.0.0) keeps the callback off the LAN, and never "localhost" so the
// registered and runtime hostnames match exactly and IPv4/IPv6 resolution is
// unambiguous.
const loopbackHost = "127.0.0.1"

// defaultAuthBase is the production authorization server. Its issuer/origin is
// the hard-pin: discovery may only confirm paths within it, never repoint the
// flow to another origin.
const defaultAuthBase = "https://auth.deploys.app"

// bakedInClientID returns a pre-provisioned public client_id for an auth base,
// or "" to self-provision via Dynamic Client Registration. No client is seeded
// today, so this is empty and the CLI registers (once, cached) against every
// auth base. An operator may later seed a public client for the default host and
// return it here to avoid per-machine DCR rows.
func bakedInClientID(authBase string) string {
	return ""
}

// errLoginTimeout is returned when the browser login does not complete in time.
// It is also the signal that a stale cached client_id may be to blame (a bad id
// is rejected at the authorize step in the browser, so the loopback never fires).
var errLoginTimeout = errors.New("login timed out")

// LoginOptions configures a browser login. Zero values are sensible defaults.
type LoginOptions struct {
	// OpenBrowser opens the authorize URL. nil uses the platform opener. Tests
	// inject a function that drives the flow against an httptest server.
	OpenBrowser func(url string) error
	// NoBrowser prints the URL instead of opening it.
	NoBrowser bool
	// Port pins the loopback port (for a fixed SSH forward). 0 = random.
	Port int
	// Timeout bounds the wait for the browser callback. 0 = 3 minutes.
	Timeout time.Duration
	// Stderr receives progress (the authorize URL, SSH guidance). nil discards.
	Stderr io.Writer
}

// Result is a completed login: the bearer token and its lifetime.
type Result struct {
	Token     string
	ExpiresIn time.Duration
}

type metadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RegistrationEndpoint  string `json:"registration_endpoint"`
	RevocationEndpoint    string `json:"revocation_endpoint"`
}

// Login runs the full Authorization Code + PKCE + loopback browser flow against
// authBase and returns the minted bearer. apiEndpoint is unused by the flow
// itself (the caller resolves identity with it afterwards) but is accepted so the
// signature reads end-to-end. On a timeout with a cached client_id, the client is
// re-registered once and the flow retried (a stale id otherwise hangs silently).
func Login(ctx context.Context, authBase, apiEndpoint string, opts LoginOptions) (Result, error) {
	authBase = strings.TrimRight(authBase, "/")
	if err := validateBase(authBase); err != nil {
		return Result{}, err
	}
	meta, err := discoverMetadata(ctx, authBase)
	if err != nil {
		return Result{}, err
	}

	clientID, fromCache, err := ensureClientID(ctx, authBase, meta.RegistrationEndpoint)
	if err != nil {
		return Result{}, err
	}

	res, err := runFlow(ctx, meta, clientID, opts)
	if errors.Is(err, errLoginTimeout) && fromCache {
		// A stale cached client_id is rejected in the browser at the authorize
		// step, so the loopback never receives a callback and we only see a
		// timeout. Re-register once and retry before giving up.
		newID, rerr := registerClient(ctx, meta.RegistrationEndpoint)
		if rerr == nil {
			_ = SaveClientID(authBase, newID)
			res, err = runFlow(ctx, meta, newID, opts)
		}
	}
	if err != nil {
		return Result{}, err
	}
	return res, nil
}

// validateBase enforces https for any non-loopback auth/api base. A cleartext
// non-loopback base would leak the 7-day bearer, so it is rejected outright.
func validateBase(base string) error {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid endpoint %q", base)
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopbackHostname(u.Hostname()) {
			return nil
		}
	}
	return fmt.Errorf("endpoint %q must use https", base)
}

func isLoopbackHostname(h string) bool {
	return h == "127.0.0.1" || h == "::1" || h == "localhost"
}

// discoverMetadata fetches RFC 8414 metadata and pins it: the issuer must equal
// the configured base and every endpoint must be same-origin as the issuer, so a
// metadata doc can never repoint the token/registration endpoint at an
// exfiltration host. Discovery only locates paths within the already-trusted
// origin.
func discoverMetadata(ctx context.Context, authBase string) (metadata, error) {
	var m metadata
	u := authBase + "/.well-known/oauth-authorization-server"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return m, err
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		return m, fmt.Errorf("discover auth server: %w", err)
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		return m, fmt.Errorf("discover auth server: status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m); err != nil {
		return m, fmt.Errorf("discover auth server: %w", err)
	}

	if strings.TrimRight(m.Issuer, "/") != authBase {
		return m, fmt.Errorf("auth server issuer %q does not match %q", m.Issuer, authBase)
	}
	for _, ep := range []string{m.AuthorizationEndpoint, m.TokenEndpoint, m.RegistrationEndpoint, m.RevocationEndpoint} {
		if ep == "" {
			continue
		}
		if !sameOrigin(ep, m.Issuer) {
			return m, fmt.Errorf("auth server endpoint %q is not same-origin as issuer %q", ep, m.Issuer)
		}
	}
	if m.AuthorizationEndpoint == "" || m.TokenEndpoint == "" {
		return m, errors.New("auth server metadata missing authorization or token endpoint")
	}
	return m, nil
}

func sameOrigin(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return ua.Scheme == ub.Scheme && strings.EqualFold(ua.Host, ub.Host)
}

// ensureClientID returns a usable client_id: a baked-in one, the cached DCR id,
// or a freshly registered one (cached for reuse). fromCache reports whether the
// id came from the cache, so a timeout can trigger a single re-registration.
func ensureClientID(ctx context.Context, authBase, regEndpoint string) (clientID string, fromCache bool, err error) {
	if id := bakedInClientID(authBase); id != "" {
		return id, false, nil
	}
	if id, ok, lerr := LoadClientID(authBase); lerr != nil {
		return "", false, lerr
	} else if ok {
		return id, true, nil
	}
	if regEndpoint == "" {
		return "", false, errors.New("auth server does not advertise a registration endpoint")
	}
	id, rerr := registerClient(ctx, regEndpoint)
	if rerr != nil {
		return "", false, rerr
	}
	if serr := SaveClientID(authBase, id); serr != nil {
		return "", false, serr
	}
	return id, false, nil
}

// registerClient performs Dynamic Client Registration (RFC 7591) for a public
// client with the port-less loopback redirect.
func registerClient(ctx context.Context, regEndpoint string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"client_name":                "deploys-cli",
		"redirect_uris":              []string{loopbackRedirect},
		"grant_types":                []string{"authorization_code"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, regEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("register client: %w", err)
	}
	defer drain(resp)
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("register client: status %d", resp.StatusCode)
	}
	var out struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return "", fmt.Errorf("register client: %w", err)
	}
	if out.ClientID == "" {
		return "", errors.New("register client: empty client_id in response")
	}
	return out.ClientID, nil
}

// runFlow starts the loopback listener, drives the browser, captures the code,
// and exchanges it for a token.
func runFlow(ctx context.Context, meta metadata, clientID string, opts LoginOptions) (Result, error) {
	verifier, challenge := generatePKCE()
	state := generateState()

	addr := fmt.Sprintf("%s:%d", loopbackHost, opts.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if opts.Port != 0 {
			return Result{}, fmt.Errorf("port %d is in use; omit -port for a random port", opts.Port)
		}
		return Result{}, err
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://%s:%d/callback", loopbackHost, port)

	authURL := buildAuthorizeURL(meta.AuthorizationEndpoint, clientID, state, redirectURI, challenge)

	stderr := opts.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	code, err := captureCode(ctx, ln, state, opts, authURL, port, stderr)
	if err != nil {
		return Result{}, err
	}

	tok, err := exchangeCode(ctx, meta.TokenEndpoint, clientID, code, verifier, redirectURI)
	if err != nil {
		return Result{}, err
	}
	return tok, nil
}

// captureCode opens the browser, serves exactly one /callback on the loopback
// listener, validates state, and returns the authorization code.
func captureCode(ctx context.Context, ln net.Listener, state string, opts LoginOptions, authURL string, port int, stderr io.Writer) (string, error) {
	done := make(chan struct {
		code string
		err  error
	}, 1)
	var once sync.Once
	finish := func(code string, err error) {
		once.Do(func() {
			done <- struct {
				code string
				err  error
			}{code, err}
		})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		setLoopbackHeaders(w)
		q := r.URL.Query()
		// Validate state first (constant-time): the callback is reachable by any
		// local process, so a forged callback must be rejected.
		if subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1 {
			writeErrorPage(w)
			finish("", errors.New("login failed: state mismatch"))
			return
		}
		if e := q.Get("error"); e != "" {
			writeErrorPage(w)
			finish("", fmt.Errorf("authorization denied: %s", sanitizeOAuthError(e)))
			return
		}
		code := q.Get("code")
		if code == "" {
			writeErrorPage(w)
			finish("", errors.New("login failed: missing authorization code"))
			return
		}
		writeSuccessPage(w)
		finish(code, nil)
	})
	// Everything else 404s and never keeps the port alive or reflects input.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	openOrPrint(opts, authURL, port, stderr)

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Minute
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-done:
		return res.code, res.err
	case <-timer.C:
		return "", errLoginTimeout
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// openOrPrint opens the browser (or prints the URL) and always echoes the
// authorize URL so a headless box where the opener silently no-ops still gives
// the user something to copy. With NoBrowser it also prints the SSH forward hint.
func openOrPrint(opts LoginOptions, authURL string, port int, stderr io.Writer) {
	open := opts.OpenBrowser
	if open == nil {
		open = defaultOpenBrowser
	}
	if !opts.NoBrowser {
		fmt.Fprintln(stderr, "Opening your browser to sign in...")
		if err := open(authURL); err != nil {
			opts.NoBrowser = true
		}
	}
	fmt.Fprintf(stderr, "If your browser did not open, visit:\n  %s\n", authURL)
	if opts.NoBrowser {
		fmt.Fprintf(stderr,
			"\nOn a remote/SSH host, forward the callback port from your local machine:\n"+
				"  ssh -L %d:127.0.0.1:%d <this-host>\n"+
				"then open the URL above in your local browser.\n", port, port)
	}
}

func buildAuthorizeURL(authzEndpoint, clientID, state, redirectURI, challenge string) string {
	p := url.Values{}
	p.Set("client_id", clientID)
	p.Set("response_type", "code")
	p.Set("state", state)
	p.Set("redirect_uri", redirectURI)
	p.Set("code_challenge", challenge)
	p.Set("code_challenge_method", "S256")
	sep := "?"
	if strings.Contains(authzEndpoint, "?") {
		sep = "&"
	}
	return authzEndpoint + sep + p.Encode()
}

// exchangeCode swaps the authorization code for a token at the token endpoint.
func exchangeCode(ctx context.Context, tokenEndpoint, clientID, code, verifier, redirectURI string) (Result, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	form.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient().Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("token exchange: %w", err)
	}
	defer drain(resp)

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		var oe struct {
			Error string `json:"error"`
			Desc  string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &oe)
		if oe.Error != "" {
			return Result{}, fmt.Errorf("token exchange failed: %s", strings.TrimSpace(oe.Error+" "+oe.Desc))
		}
		return Result{}, fmt.Errorf("token exchange failed: status %d", resp.StatusCode)
	}

	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return Result{}, fmt.Errorf("token exchange: %w", err)
	}
	if out.AccessToken == "" {
		return Result{}, errors.New("token exchange: empty access_token")
	}
	exp := time.Duration(out.ExpiresIn) * time.Second
	if exp <= 0 {
		exp = 7 * 24 * time.Hour
	}
	return Result{Token: out.AccessToken, ExpiresIn: exp}, nil
}

func generatePKCE() (verifier, challenge string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}

func generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// sanitizeOAuthError whitelists a server-supplied OAuth error code to a short
// known set so attacker-influenced callback content is never echoed verbatim.
func sanitizeOAuthError(code string) string {
	switch code {
	case "access_denied", "invalid_request", "invalid_scope", "server_error",
		"temporarily_unavailable", "unauthorized_client", "unsupported_response_type":
		return code
	default:
		return "authorization error"
	}
}

func defaultOpenBrowser(u string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{u}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", u}
	default:
		name, args = "xdg-open", []string{u}
	}
	return exec.Command(name, args...).Start()
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func drain(resp *http.Response) {
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}
