// Package auth implements the deploys CLI's interactive browser login and the
// multi-account credential store that backs it.
//
// The store is a 0600 JSON file under the user's config dir. Accounts are keyed
// by (api endpoint, email) so the same person can hold separate sessions per
// endpoint (prod, staging, self-hosted) without collision, and a per-endpoint
// "active" pointer selects the default account for each endpoint. A separate
// 0600 file caches the dynamically-registered OAuth client_id per auth server.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// schemaVersion is the on-disk credentials schema version. A file written by a
// newer CLI (higher version) is rejected rather than silently downgraded, so an
// old binary never clobbers a format it does not understand.
const schemaVersion = 1

// keySep joins the normalized endpoint and the email into an account key. It is
// a NUL byte: invisible in rendered JSON and impossible to occur inside a URL or
// an email address, so the two halves can never be confused.
const keySep = "\x00"

// defaultEndpoint mirrors the api client's default base; normalizeEndpoint must
// agree with client.Client.endpoint() so a stored key matches the endpoint the
// client actually talks to.
const defaultEndpoint = "https://api.deploys.app/"

// Account is one stored login: the 7-day bearer token plus the identity and the
// endpoints needed to use it and later revoke it.
type Account struct {
	Key          string    `json:"key"`          // normalizeEndpoint(endpoint) + keySep + email
	Endpoint     string    `json:"endpoint"`     // normalized api base the token is used against
	AuthEndpoint string    `json:"authEndpoint"` // auth base that minted the token (logout/revoke)
	Email        string    `json:"email"`        // identity from me.get
	Token        string    `json:"token"`        // "deploys-api." bearer (0600 protects it)
	ExpiresAt    time.Time `json:"expiresAt"`    // issuedAt + expires_in (~7d)
	IssuedAt     time.Time `json:"issuedAt"`
}

// Expired reports whether the token is at or past its expiry. A zero ExpiresAt
// is treated as never-expiring (it should not happen for a real login).
func (a Account) Expired() bool {
	return !a.ExpiresAt.IsZero() && !time.Now().Before(a.ExpiresAt)
}

// Credentials is the whole on-disk account store.
type Credentials struct {
	Version  int               `json:"version"`
	Active   map[string]string `json:"active,omitempty"` // normalized endpoint -> active account key
	Accounts []Account         `json:"accounts"`
}

// AccountKey builds the storage key for an (endpoint, email) pair. The endpoint
// is normalized so "https://api.deploys.app" and "https://api.deploys.app/"
// collide to one key.
func AccountKey(endpoint, email string) string {
	return normalizeEndpoint(endpoint) + keySep + email
}

// normalizeEndpoint mirrors the api client's endpoint() normalization: an empty
// value becomes the default base, and the result always ends in a single "/".
func normalizeEndpoint(ep string) string {
	if ep == "" {
		return defaultEndpoint
	}
	return strings.TrimSuffix(ep, "/") + "/"
}

// NormalizeEndpoint is the exported form of normalizeEndpoint, for callers that
// need the same normalized api base the store keys on.
func NormalizeEndpoint(ep string) string { return normalizeEndpoint(ep) }

// Find returns the account stored under key.
func (c *Credentials) Find(key string) (*Account, bool) {
	for i := range c.Accounts {
		if c.Accounts[i].Key == key {
			return &c.Accounts[i], true
		}
	}
	return nil, false
}

// Upsert replaces the account with the same key, or appends it.
func (c *Credentials) Upsert(a Account) {
	for i := range c.Accounts {
		if c.Accounts[i].Key == a.Key {
			c.Accounts[i] = a
			return
		}
	}
	c.Accounts = append(c.Accounts, a)
}

// Remove deletes the account under key and clears any active pointer that
// referenced it. If another account remains on the same endpoint, the active
// pointer for that endpoint is repointed at it; otherwise the pointer is
// dropped. It reports whether an account was removed.
func (c *Credentials) Remove(key string) bool {
	idx := -1
	for i := range c.Accounts {
		if c.Accounts[i].Key == key {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	endpoint := c.Accounts[idx].Endpoint
	c.Accounts = append(c.Accounts[:idx], c.Accounts[idx+1:]...)

	if c.Active[endpoint] == key {
		delete(c.Active, endpoint)
		// repoint to any remaining account on the same endpoint
		for i := range c.Accounts {
			if c.Accounts[i].Endpoint == endpoint {
				if c.Active == nil {
					c.Active = map[string]string{}
				}
				c.Active[endpoint] = c.Accounts[i].Key
				break
			}
		}
	}
	return true
}

// SetActive points the given endpoint at the account key.
func (c *Credentials) SetActive(endpoint, key string) {
	if c.Active == nil {
		c.Active = map[string]string{}
	}
	c.Active[normalizeEndpoint(endpoint)] = key
}

// ActiveKey returns the active account key for an endpoint.
func (c *Credentials) ActiveKey(endpoint string) (string, bool) {
	k, ok := c.Active[normalizeEndpoint(endpoint)]
	return k, ok && k != ""
}

// ConfigDir resolves the directory holding the credential and client-id files,
// in order: DEPLOYS_CONFIG_DIR, then XDG_CONFIG_HOME/deploys (honored even on
// macOS, which os.UserConfigDir ignores), then os.UserConfigDir()/deploys.
func ConfigDir() (string, error) {
	if d := os.Getenv("DEPLOYS_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "deploys"), nil
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "deploys"), nil
}

func credentialsPath() (string, error) {
	d, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "credentials.json"), nil
}

func clientsPath() (string, error) {
	d, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "clients.json"), nil
}

// Load reads the credential store. A missing file yields an empty store (first
// run). A malformed file or one written by a newer CLI is an error — never
// silently treated as empty, which would discard every stored login.
func Load() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}
	b, err := readFileSecure(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Credentials{Version: schemaVersion, Active: map[string]string{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("credentials file %s is corrupt: %w (inspect or remove it)", path, err)
	}
	if c.Version > schemaVersion {
		return nil, fmt.Errorf("credentials file %s is version %d; upgrade the deploys cli", path, c.Version)
	}
	if c.Active == nil {
		c.Active = map[string]string{}
	}
	return &c, nil
}

// Save writes the store atomically with 0600 permissions. Callers that
// read-modify-write should use Mutate, which holds the store lock across the
// whole cycle; Save itself does not lock.
func (c *Credentials) Save() error {
	c.Version = schemaVersion
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := prepareDir(filepath.Dir(path)); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, b)
}

// Mutate runs fn against the store under an exclusive lock, then saves. It is
// the only safe way to change the store: a concurrent login/logout waits rather
// than clobbering.
func Mutate(fn func(*Credentials) error) error {
	unlock, err := acquireLock("credentials.json.lock")
	if err != nil {
		return err
	}
	defer unlock()

	c, err := Load()
	if err != nil {
		return err
	}
	if err := fn(c); err != nil {
		return err
	}
	return c.Save()
}

// clientCache is the on-disk DCR client_id cache, keyed by auth base URL.
type clientCache struct {
	Version int               `json:"version"`
	Clients map[string]string `json:"clients"`
}

// LoadClientID returns the cached OAuth client_id for an auth base.
func LoadClientID(authBase string) (string, bool, error) {
	path, err := clientsPath()
	if err != nil {
		return "", false, err
	}
	b, err := readFileSecure(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	var cc clientCache
	if err := json.Unmarshal(b, &cc); err != nil {
		// A corrupt client cache is recoverable (we can re-register), so treat it
		// as empty rather than failing the whole login.
		return "", false, nil
	}
	id, ok := cc.Clients[authBase]
	return id, ok && id != "", nil
}

// SaveClientID caches the OAuth client_id for an auth base.
func SaveClientID(authBase, clientID string) error {
	unlock, err := acquireLock("clients.json.lock")
	if err != nil {
		return err
	}
	defer unlock()

	path, err := clientsPath()
	if err != nil {
		return err
	}
	if err := prepareDir(filepath.Dir(path)); err != nil {
		return err
	}
	var cc clientCache
	if b, rerr := readFileSecure(path); rerr == nil {
		_ = json.Unmarshal(b, &cc)
	} else if !errors.Is(rerr, os.ErrNotExist) {
		return rerr
	}
	if cc.Clients == nil {
		cc.Clients = map[string]string{}
	}
	cc.Version = schemaVersion
	cc.Clients[authBase] = clientID
	b, err := json.MarshalIndent(&cc, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, b)
}

// prepareDir creates the config dir 0700 and, on unix, refuses to proceed if it
// is group/other-writable and cannot be tightened, or is owned by another user.
// The whole at-rest model rests on a private parent dir.
func prepareDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if fi.Mode().Perm()&0o077 != 0 {
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("config dir %s is group/other-writable and could not be tightened: %w", dir, err)
		}
	}
	owned, err := ownedByCurrentUser(fi)
	if err != nil {
		return err
	}
	if !owned {
		return fmt.Errorf("refusing to use config dir %s: it is owned by another user", dir)
	}
	return nil
}

// writeFileAtomic writes data to path via a uniquely-named temp file in the same
// dir (created O_EXCL with mode 0600, so a hostile pre-created symlink at a
// predictable name cannot be clobbered and the mode is set at create time, not
// by a later chmod), fsync'd, then renamed over the target. os.Rename replaces
// atomically, so a reader never sees a torn file.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp := filepath.Join(dir, "."+filepath.Base(path)+"."+randHex(8)+".tmp")
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	werr := func() error {
		if _, err := f.Write(data); err != nil {
			return err
		}
		return f.Sync()
	}()
	cerr := f.Close()
	if werr != nil {
		os.Remove(tmp)
		return werr
	}
	if cerr != nil {
		os.Remove(tmp)
		return cerr
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// readFileSecure reads a 0600 store file, refusing a symlink or a file owned by
// another user, and tightening a loose-but-owned file. Permission/ownership
// checks are unix-only (os.Chmod only toggles the read-only bit on Windows).
func readFileSecure(path string) ([]byte, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to read %s: it is a symlink", path)
	}
	if runtime.GOOS != "windows" {
		owned, oerr := ownedByCurrentUser(fi)
		if oerr != nil {
			return nil, oerr
		}
		if !owned {
			return nil, fmt.Errorf("refusing to read %s: it is owned by another user", path)
		}
		if fi.Mode().Perm()&0o077 != 0 {
			if err := os.Chmod(path, 0o600); err != nil {
				return nil, fmt.Errorf("%s is too permissive and could not be tightened: %w", path, err)
			}
		}
	}
	return os.ReadFile(path)
}

// acquireLock takes an exclusive O_EXCL lockfile (portable, unlike flock, which
// Go cannot do cross-platform). A lockfile older than lockStale — or whose
// recorded PID matches the current process from a crashed prior run — is broken
// so an interrupted command can't wedge the store forever.
func acquireLock(name string) (func(), error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(dir, name)
	const lockStale = 30 * time.Second
	deadline := time.Now().Add(10 * time.Second)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			fmt.Fprintf(f, "%d %d\n", os.Getpid(), time.Now().Unix())
			f.Close()
			return func() { os.Remove(lockPath) }, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		if lockIsStale(lockPath, lockStale) {
			os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("the credential store is locked (%s); another deploys process may be running — remove the lock file if not", lockPath)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// lockIsStale reports whether a held lockfile is old enough to reclaim.
func lockIsStale(lockPath string, maxAge time.Duration) bool {
	fi, err := os.Stat(lockPath)
	if err != nil {
		// Vanished between OpenFile and Stat: treat as stale so the loop retries.
		return true
	}
	return time.Since(fi.ModTime()) > maxAge
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing is catastrophic and not something a CLI can paper
		// over; fall back to a fixed marker so the O_EXCL create still guards us.
		return "tmp"
	}
	return hex.EncodeToString(b)
}
