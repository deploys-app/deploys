package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Revoke best-effort revokes a token at the auth server's /revoke endpoint. The
// endpoint is unauthenticated and returns ok unconditionally, so a 2xx confirms
// only that the request reached a server — not that this specific token existed.
// A network/non-2xx failure is returned so the caller can keep the local entry
// rather than orphan a still-valid token.
func Revoke(ctx context.Context, authBase, token string) error {
	authBase = strings.TrimRight(authBase, "/")
	if authBase == "" {
		authBase = defaultAuthBase
	}
	if err := validateBase(authBase); err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"token": token})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authBase+"/revoke", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("revoke: %w", err)
	}
	defer drain(resp)
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("revoke: status %d", resp.StatusCode)
	}
	return nil
}
