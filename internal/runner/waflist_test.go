package runner

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// An -entries-file is one entry per line: blank lines and #-comments
// (whole-line or trailing) are stripped, surrounding whitespace trimmed.
func TestParseWAFListEntries(t *testing.T) {
	in := strings.Join([]string{
		"# office ranges",
		"203.0.113.0/24",
		"",
		"  198.51.100.7  # bastion",
		"\t2001:db8::/48\t",
		"   ",
		"#",
	}, "\n")
	want := []string{"203.0.113.0/24", "198.51.100.7", "2001:db8::/48"}
	got := parseWAFListEntries(in)
	if !slices.Equal(got, want) {
		t.Errorf("parseWAFListEntries = %q; want %q", got, want)
	}

	if got := parseWAFListEntries(""); got != nil {
		t.Errorf("parseWAFListEntries(empty) = %q; want nil", got)
	}
}

// Group help must render without touching the API, for the canonical name and
// the lowercase alias alike.
func TestWAFListHelpNeedsNoAPI(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "help")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()

	for _, args := range [][]string{
		{"wafList"},
		{"wafList", "help"},
		{"waflist", "-h"},
	} {
		if err := tmp.Truncate(0); err != nil {
			t.Fatal(err)
		}
		if _, err := tmp.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		rn := Runner{Output: tmp} // API intentionally nil
		if err := rn.Run(args...); err != nil {
			t.Errorf("Run(%v) error: %v", args, err)
			continue
		}
		b, err := os.ReadFile(tmp.Name())
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "named IP lists") {
			t.Errorf("Run(%v) output missing group description:\n%s", args, b)
		}
	}
}
