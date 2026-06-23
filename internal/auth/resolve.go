package auth

import "fmt"

// AuthRequiredError signals that the caller has no usable credential and must
// log in (or that an explicitly-named account does not resolve). main maps it to
// exit code 4 and prints its message verbatim. Its presence — not an HTTP status
// — is how the CLI distinguishes "you need to log in" from other failures.
type AuthRequiredError struct{ Msg string }

func (e *AuthRequiredError) Error() string { return e.Msg }

// Resolve picks the stored account for an endpoint, honoring selector precedence
// (an explicit -account/DEPLOYS_ACCOUNT email, else the endpoint's active
// account). explicit reports whether the selector was set explicitly, which
// changes the failure mode:
//
//   - found and unexpired               -> (account, nil)
//   - implicit and nothing usable       -> (nil, nil)            caller falls through to ADC
//   - implicit active account expired   -> (account{expired}, nil) caller warns + falls through
//   - explicit but missing or expired   -> (nil, *AuthRequiredError) caller hard-errors (exit 4)
//
// An expired account is reported via the returned expired flag; its token is
// never returned (no refresh grant exists, so a dead token must not be sent).
func Resolve(selector, endpoint string, explicit bool) (acct *Account, expired bool, err error) {
	c, lerr := Load()
	if lerr != nil {
		return nil, false, lerr
	}
	ep := normalizeEndpoint(endpoint)

	var key string
	switch {
	case selector != "":
		key = AccountKey(ep, selector)
	default:
		if ak, ok := c.ActiveKey(ep); ok {
			key = ak
		}
	}

	if key == "" {
		if explicit {
			return nil, false, notFound(selector, ep)
		}
		return nil, false, nil
	}

	a, ok := c.Find(key)
	if !ok {
		if explicit {
			return nil, false, notFound(selector, ep)
		}
		return nil, false, nil
	}
	if a.Expired() {
		if explicit {
			return nil, true, &AuthRequiredError{Msg: fmt.Sprintf(
				"stored session for %s on %s has expired. Run 'deploys login' to sign in again.", a.Email, ep)}
		}
		return a, true, nil
	}
	return a, false, nil
}

// Lookup returns the selected/active account for an endpoint regardless of
// expiry, for display by `deploys auth status`/`token`. A nil account with a nil
// error means there is no stored account for the endpoint.
func Lookup(selector, endpoint string) (*Account, error) {
	c, err := Load()
	if err != nil {
		return nil, err
	}
	ep := normalizeEndpoint(endpoint)
	var key string
	if selector != "" {
		key = AccountKey(ep, selector)
	} else if ak, ok := c.ActiveKey(ep); ok {
		key = ak
	}
	if key == "" {
		return nil, nil
	}
	a, ok := c.Find(key)
	if !ok {
		return nil, nil
	}
	return a, nil
}

func notFound(selector, endpoint string) error {
	if selector != "" {
		return &AuthRequiredError{Msg: fmt.Sprintf(
			"no stored login for %s on %s. Run 'deploys login' to sign in.", selector, endpoint)}
	}
	return &AuthRequiredError{Msg: fmt.Sprintf(
		"no stored login for %s. Run 'deploys login' to sign in.", endpoint)}
}
