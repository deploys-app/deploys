//go:build windows

package auth

import "os"

// ownedByCurrentUser is a no-op on Windows, where POSIX uid ownership does not
// apply and os.Chmod only toggles the read-only bit. File ACLs are not enforced
// here; this is documented as a best-effort limitation.
func ownedByCurrentUser(os.FileInfo) (bool, error) {
	return true, nil
}
