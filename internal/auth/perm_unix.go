//go:build !windows

package auth

import (
	"os"
	"syscall"
)

// ownedByCurrentUser reports whether the file is owned by the current uid. If
// the platform doesn't expose a uid (it always does on unix), it defers to true.
func ownedByCurrentUser(fi os.FileInfo) (bool, error) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return true, nil
	}
	return int(st.Uid) == os.Getuid(), nil
}
