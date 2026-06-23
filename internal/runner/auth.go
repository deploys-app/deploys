package runner

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/deploys-app/api"
	"github.com/deploys-app/api/client"

	"github.com/deploys-app/deploys/internal/auth"
)

// doLogin is the login entry point, indirected so tests can stub the browser
// flow and assert the runner's persistence/identity wiring without a network.
var doLogin = auth.Login

// authBaseURL resolves the authorization server base: DEPLOYS_AUTH_ENDPOINT, or
// the production default. It is independent of the api endpoint.
func authBaseURL() string {
	if b := os.Getenv("DEPLOYS_AUTH_ENDPOINT"); b != "" {
		return b
	}
	return "https://auth.deploys.app"
}

// apiEndpointFor resolves the api endpoint for an auth command: the -endpoint
// flag, else DEPLOYS_ENDPOINT.
func apiEndpointFor(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	return os.Getenv("DEPLOYS_ENDPOINT")
}

// selector returns the account selector: the global -account (pre-scanned in
// main), else DEPLOYS_ACCOUNT.
func (rn Runner) selector() string {
	if rn.Account != "" {
		return rn.Account
	}
	return os.Getenv("DEPLOYS_ACCOUNT")
}

func (rn Runner) login(args ...string) error {
	if len(args) > 0 && IsHelpArg(args[0]) {
		writeLoginUsage(rn.output())
		return nil
	}
	f := flag.NewFlagSet("deploys login", flag.ExitOnError)
	f.SetOutput(rn.output())
	f.Usage = func() { writeLoginUsage(rn.output()) }
	var (
		endpoint  string
		noBrowser bool
		port      int
		timeout   time.Duration
	)
	f.StringVar(&endpoint, "endpoint", "", "api endpoint for this account (default $DEPLOYS_ENDPOINT or https://api.deploys.app/)")
	f.BoolVar(&noBrowser, "no-browser", false, "print the authorization URL instead of opening a browser")
	f.IntVar(&port, "port", 0, "loopback callback port for a fixed SSH forward (default: a random free port)")
	f.DurationVar(&timeout, "timeout", 3*time.Minute, "how long to wait for the browser login")
	if err := f.Parse(args); err != nil {
		return err
	}

	apiEndpoint := apiEndpointFor(endpoint)
	authBase := authBaseURL()

	res, err := doLogin(context.Background(), authBase, apiEndpoint, auth.LoginOptions{
		NoBrowser: noBrowser,
		Port:      port,
		Timeout:   timeout,
		Stderr:    os.Stderr,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	email, resolved := meGetEmail(apiEndpoint, res.Token)
	if !resolved {
		email = placeholderEmail(res.Token)
	}

	acct := auth.Account{
		Key:          auth.AccountKey(apiEndpoint, email),
		Endpoint:     auth.NormalizeEndpoint(apiEndpoint),
		AuthEndpoint: authBase,
		Email:        email,
		Token:        res.Token,
		ExpiresAt:    now.Add(res.ExpiresIn),
		IssuedAt:     now,
	}
	if err := auth.Mutate(func(c *auth.Credentials) error {
		c.Upsert(acct)
		c.SetActive(apiEndpoint, acct.Key)
		return nil
	}); err != nil {
		return err
	}

	if resolved {
		fmt.Fprintf(rn.output(), "Logged in as %s (now active)\n", email)
	} else {
		fmt.Fprintln(os.Stderr, "warning: logged in, but could not resolve your email (me.get failed); "+
			"stored under the token only — run 'deploys auth status' once online to backfill identity")
		fmt.Fprintln(rn.output(), "Logged in (identity unresolved; now active)")
	}
	return nil
}

// meGetEmail resolves the authenticated identity for a freshly minted token. It
// builds its own one-off client (rn.API is nil for auth commands) with the same
// 15s timeout newAPIClient uses, so a hung me.get can't strand the token.
func meGetEmail(endpoint, token string) (string, bool) {
	c := &client.Client{
		Endpoint:   endpoint,
		Channel:    api.AuditChannelCLI,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		Auth: func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+token)
		},
	}
	res, err := c.Me().Get(context.Background(), &api.Empty{})
	if err != nil || res == nil || res.Email == "" {
		return "", false
	}
	return res.Email, true
}

// placeholderEmail derives a stable, addressable identity for an account whose
// me.get failed, so the entry is still revocable by key.
func placeholderEmail(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "unresolved-" + hex.EncodeToString(sum[:4]) + "@local"
}

func (rn Runner) logout(args ...string) error {
	if len(args) > 0 && IsHelpArg(args[0]) {
		writeLogoutUsage(rn.output())
		return nil
	}
	f := flag.NewFlagSet("deploys logout", flag.ExitOnError)
	f.SetOutput(rn.output())
	f.Usage = func() { writeLogoutUsage(rn.output()) }
	var (
		endpoint string
		all      bool
		yes      bool
	)
	f.StringVar(&endpoint, "endpoint", "", "api endpoint of the account to log out (default $DEPLOYS_ENDPOINT)")
	f.BoolVar(&all, "all", false, "remove every stored account")
	f.BoolVar(&yes, "yes", false, "skip the confirmation prompt (required for -all without a terminal)")
	if err := f.Parse(args); err != nil {
		return err
	}

	if all {
		return rn.logoutAll(yes)
	}

	apiEndpoint := apiEndpointFor(endpoint)
	acct, err := auth.Lookup(rn.selector(), apiEndpoint)
	if err != nil {
		return err
	}
	if acct == nil {
		fmt.Fprintln(rn.output(), "nothing to log out")
		return nil
	}

	// Best-effort server-side revoke against the auth base that minted the token.
	// On failure, keep the entry and fail loudly — never orphan a live token.
	if rerr := auth.Revoke(context.Background(), acct.AuthEndpoint, acct.Token); rerr != nil {
		return fmt.Errorf("token NOT revoked server-side (%v); it remains valid until %s. "+
			"Re-run 'deploys logout' when online", rerr, acct.ExpiresAt.Format(time.RFC3339))
	}

	norm := auth.NormalizeEndpoint(apiEndpoint)
	var reassigned string
	if err := auth.Mutate(func(c *auth.Credentials) error {
		c.Remove(acct.Key)
		if k, ok := c.ActiveKey(norm); ok {
			if a, ok := c.Find(k); ok {
				reassigned = a.Email
			}
		}
		return nil
	}); err != nil {
		return err
	}

	fmt.Fprintf(rn.output(), "Logged out %s (removed; revoke acknowledged by %s)\n", acct.Email, acct.AuthEndpoint)
	if reassigned != "" {
		fmt.Fprintf(rn.output(), "Active account for %s is now %s\n", norm, reassigned)
	}
	return nil
}

func (rn Runner) logoutAll(yes bool) error {
	c, err := auth.Load()
	if err != nil {
		return err
	}
	if len(c.Accounts) == 0 {
		fmt.Fprintln(rn.output(), "nothing to log out")
		return nil
	}
	if !yes {
		if !isTTY(os.Stdin) {
			return fmt.Errorf("refusing to remove all accounts without -yes (no interactive terminal)")
		}
		fmt.Fprintf(rn.output(), "Remove all %d stored account(s)? [y/N]: ", len(c.Accounts))
		if !readYes(os.Stdin) {
			fmt.Fprintln(rn.output(), "aborted")
			return nil
		}
	}

	var revokedKeys []string
	var failed []string
	for _, a := range c.Accounts {
		if rerr := auth.Revoke(context.Background(), a.AuthEndpoint, a.Token); rerr != nil {
			failed = append(failed, a.Email)
			continue
		}
		revokedKeys = append(revokedKeys, a.Key)
	}
	if err := auth.Mutate(func(cc *auth.Credentials) error {
		for _, k := range revokedKeys {
			cc.Remove(k)
		}
		return nil
	}); err != nil {
		return err
	}

	fmt.Fprintf(rn.output(), "Removed %d account(s)\n", len(revokedKeys))
	if len(failed) > 0 {
		return fmt.Errorf("could not revoke %d account(s) (kept locally): %s — re-run when online",
			len(failed), strings.Join(failed, ", "))
	}
	return nil
}

func (rn Runner) authGroup(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("auth")
	}
	switch args[0] {
	default:
		return rn.unknownSub("auth", args[0])
	case "login":
		return rn.login(args[1:]...)
	case "logout":
		return rn.logout(args[1:]...)
	case "status":
		return rn.authStatus(args[1:]...)
	case "list":
		return rn.authList(args[1:]...)
	case "switch":
		return rn.authSwitch(args[1:]...)
	case "token":
		return rn.authToken(args[1:]...)
	}
}

// authStatusItem is the structured form of `deploys auth status` for -ojson/yaml.
// It carries only the in-effect credential's facts (no KYC, no token).
type authStatusItem struct {
	Source    string     `json:"source" yaml:"source"`
	Endpoint  string     `json:"endpoint" yaml:"endpoint"`
	Account   string     `json:"account" yaml:"account"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty" yaml:"expiresAt,omitempty"`
}

func (rn Runner) authStatus(args ...string) error {
	f := rn.subFlagSet("auth", "status")
	var endpoint string
	f.StringVar(&endpoint, "endpoint", "", "api endpoint to report (default $DEPLOYS_ENDPOINT)")
	if err := f.Parse(args); err != nil {
		return err
	}
	apiEndpoint := apiEndpointFor(endpoint)
	norm := auth.NormalizeEndpoint(apiEndpoint)

	var item authStatusItem
	item.Endpoint = norm
	expiresLine := "(n/a)"

	switch {
	case os.Getenv("DEPLOYS_AUTH_USER") != "" && os.Getenv("DEPLOYS_AUTH_PASS") != "":
		item.Source = "DEPLOYS_AUTH_USER/DEPLOYS_AUTH_PASS environment (service account)"
		item.Account = os.Getenv("DEPLOYS_AUTH_USER")
		expiresLine = "(unknown — managed by the environment)"
	case os.Getenv("DEPLOYS_TOKEN") != "":
		item.Source = "DEPLOYS_TOKEN environment variable"
		item.Account = "(unknown — env token carries no identity)"
		expiresLine = "(unknown — managed by the environment)"
	default:
		acct, err := auth.Lookup(rn.selector(), apiEndpoint)
		if err != nil {
			return err
		}
		if acct == nil {
			// No env credential and no stored account; a normal command may still
			// authenticate via Google ADC (the legacy fallback), which status does
			// not probe — so phrase this honestly rather than "not logged in".
			item.Source = "no stored login (commands fall back to env vars or Google ADC)"
			item.Account = "(none — run 'deploys login')"
		} else {
			item.Source = "active account (credentials file)"
			item.Account = acct.Email
			e := acct.ExpiresAt
			item.ExpiresAt = &e
			expiresLine = formatExpiry(e)
		}
	}

	if rn.OutputMode == "" || rn.OutputMode == "table" {
		out := rn.output()
		fmt.Fprintf(out, "%-10s%s\n", "Source", item.Source)
		fmt.Fprintf(out, "%-10s%s\n", "Endpoint", item.Endpoint)
		fmt.Fprintf(out, "%-10s%s\n", "Account", item.Account)
		fmt.Fprintf(out, "%-10s%s\n", "Expires", expiresLine)
		if item.ExpiresAt != nil {
			if d := time.Until(*item.ExpiresAt); d > 0 && d < 24*time.Hour {
				fmt.Fprintf(os.Stderr, "warning: your deploys session expires in %s — run 'deploys login' to sign in again\n", humanDur(d))
			}
		}
		return nil
	}
	return rn.print(item)
}

// AuthListItem is one row of `deploys auth list`.
type AuthListItem struct {
	Active   bool   `json:"active" yaml:"active"`
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	Account  string `json:"account" yaml:"account"`
	Expires  string `json:"expires" yaml:"expires"`
}

type authListResult struct {
	Items []AuthListItem `json:"items" yaml:"items"`
}

func (a authListResult) Table() [][]string {
	rows := [][]string{{"ACTIVE", "ENDPOINT", "ACCOUNT", "EXPIRES"}}
	for _, it := range a.Items {
		active := ""
		if it.Active {
			active = "*"
		}
		rows = append(rows, []string{active, it.Endpoint, it.Account, it.Expires})
	}
	return rows
}

func (rn Runner) authList(args ...string) error {
	f := rn.subFlagSet("auth", "list")
	if err := f.Parse(args); err != nil {
		return err
	}
	c, err := auth.Load()
	if err != nil {
		return err
	}
	if len(c.Accounts) == 0 {
		fmt.Fprintln(rn.output(), "no stored accounts (run 'deploys login')")
		return nil
	}

	accts := append([]auth.Account(nil), c.Accounts...)
	sort.Slice(accts, func(i, j int) bool {
		if accts[i].Endpoint != accts[j].Endpoint {
			return accts[i].Endpoint < accts[j].Endpoint
		}
		return accts[i].Email < accts[j].Email
	})

	var items []AuthListItem
	for _, a := range accts {
		active := false
		if k, ok := c.ActiveKey(a.Endpoint); ok && k == a.Key {
			active = true
		}
		items = append(items, AuthListItem{
			Active:   active,
			Endpoint: a.Endpoint,
			Account:  a.Email,
			Expires:  formatExpiryShort(a.ExpiresAt),
		})
	}
	return rn.print(authListResult{Items: items})
}

func (rn Runner) authSwitch(args ...string) error {
	f := rn.subFlagSet("auth", "switch")
	var endpoint string
	f.StringVar(&endpoint, "endpoint", "", "api endpoint whose active account to change (default $DEPLOYS_ENDPOINT)")
	if err := f.Parse(args); err != nil {
		return err
	}
	apiEndpoint := apiEndpointFor(endpoint)
	norm := auth.NormalizeEndpoint(apiEndpoint)
	sel := rn.selector()

	c, err := auth.Load()
	if err != nil {
		return err
	}
	var onEp []auth.Account
	for _, a := range c.Accounts {
		if a.Endpoint == norm {
			onEp = append(onEp, a)
		}
	}
	if len(onEp) == 0 {
		return &auth.AuthRequiredError{Msg: fmt.Sprintf("no stored accounts for %s; run 'deploys login'", norm)}
	}

	var target *auth.Account
	if sel != "" {
		key := auth.AccountKey(apiEndpoint, sel)
		for i := range onEp {
			if onEp[i].Key == key {
				target = &onEp[i]
				break
			}
		}
		if target == nil {
			return &auth.AuthRequiredError{Msg: fmt.Sprintf("no stored login for %s on %s", sel, norm)}
		}
	} else if len(onEp) == 1 {
		target = &onEp[0]
	} else {
		var names []string
		for _, a := range onEp {
			names = append(names, a.Email)
		}
		return fmt.Errorf("multiple accounts on %s (%s); pass -account <email>", norm, strings.Join(names, ", "))
	}

	if err := auth.Mutate(func(cc *auth.Credentials) error {
		cc.SetActive(apiEndpoint, target.Key)
		return nil
	}); err != nil {
		return err
	}
	fmt.Fprintf(rn.output(), "Active account for %s is now %s\n", norm, target.Email)
	return nil
}

func (rn Runner) authToken(args ...string) error {
	f := rn.subFlagSet("auth", "token")
	var (
		endpoint string
		force    bool
	)
	f.StringVar(&endpoint, "endpoint", "", "api endpoint of the account (default $DEPLOYS_ENDPOINT)")
	f.BoolVar(&force, "force", false, "print even when stdout is a terminal")
	if err := f.Parse(args); err != nil {
		return err
	}
	apiEndpoint := apiEndpointFor(endpoint)

	// Mirror a normal command's credential precedence.
	if os.Getenv("DEPLOYS_AUTH_USER") != "" && os.Getenv("DEPLOYS_AUTH_PASS") != "" {
		return fmt.Errorf("active credential is a service-account key, not a bearer token")
	}
	var token string
	if t := os.Getenv("DEPLOYS_TOKEN"); t != "" {
		token = t
	} else {
		acct, err := auth.Lookup(rn.selector(), apiEndpoint)
		if err != nil {
			return err
		}
		if acct == nil {
			return &auth.AuthRequiredError{Msg: "not logged in. Run 'deploys login' to sign in."}
		}
		if acct.Expired() {
			return &auth.AuthRequiredError{Msg: "your deploys session has expired. Run 'deploys login' to sign in again."}
		}
		token = acct.Token
	}

	out := rn.output()
	tty := isTTY(out)
	if tty && !force {
		return fmt.Errorf("refusing to print a token to a terminal; pipe it, or pass -force")
	}
	if tty {
		fmt.Fprintln(out, token)
	} else {
		// No trailing newline so $(deploys auth token) captures the exact value.
		fmt.Fprint(out, token)
	}
	return nil
}

func isTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func readYes(f *os.File) bool {
	s := bufio.NewScanner(f)
	if !s.Scan() {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(s.Text())) {
	case "y", "yes":
		return true
	}
	return false
}

// humanDur renders a positive duration compactly (e.g. "6d 23h", "18h 4m",
// "45m", "30s").
func humanDur(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	case mins > 0:
		return fmt.Sprintf("%dm", mins)
	default:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}

func formatExpiry(t time.Time) string {
	if t.IsZero() {
		return "(unknown)"
	}
	base := t.Local().Format("2006-01-02 15:04")
	if d := time.Until(t); d > 0 {
		return base + " (in " + humanDur(d) + ")"
	}
	return base + " (expired)"
}

func formatExpiryShort(t time.Time) string {
	if t.IsZero() {
		return "(unknown)"
	}
	base := t.Local().Format("2006-01-02")
	if d := time.Until(t); d > 0 {
		return base + " (" + humanDur(d) + ")"
	}
	return base + " (expired)"
}
