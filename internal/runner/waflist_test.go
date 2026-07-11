package runner

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/deploys-app/api"
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

// buildWAFListSet's source semantics: base entries from -entries-file, else
// the -f yaml; -entry appends to the base; scalar flags override -f values;
// a source that yields no entries is an error, while explicit emptying via
// `entries: []` in -f is allowed.
func TestBuildWAFListSet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	spec := write("list.yaml", strings.Join([]string{
		"project: acme",
		"name: office-ips",
		"description: office ranges",
		"type: ip",
		"entries:",
		"  - 203.0.113.0/24",
		"  - 198.51.100.7",
		// read-only fields from `wafList get -oyaml` must not break the parse
		"referencedBy:",
		"  - gke.cluster-rcf2",
		"createdAt: 2026-07-10T00:00:00Z",
	}, "\n"))
	emptySpec := write("empty.yaml", "project: acme\nname: office-ips\nentries: []\n")
	noEntriesSpec := write("noentries.yaml", "project: acme\nname: office-ips\n")
	entriesFile := write("entries.txt", "# office\n192.0.2.0/24\n2001:db8::/48 # v6\n")
	blankEntriesFile := write("blank.txt", "# nothing here\n\n   \n")

	tests := []struct {
		name    string
		in      wafListSetFlags
		want    api.WAFListSet
		wantErr string
	}{
		{
			name: "flags only",
			in:   wafListSetFlags{project: "acme", name: "office-ips", entries: multiFlag{"203.0.113.0/24"}},
			want: api.WAFListSet{Project: "acme", Name: "office-ips", Entries: []string{"203.0.113.0/24"}},
		},
		{
			name: "spec file only",
			in:   wafListSetFlags{fn: spec},
			want: api.WAFListSet{
				Project: "acme", Name: "office-ips", Description: "office ranges", Type: api.WAFListTypeIP,
				Entries: []string{"203.0.113.0/24", "198.51.100.7"},
			},
		},
		{
			name: "entry flags append to spec entries, scalar flags override",
			in:   wafListSetFlags{fn: spec, entries: multiFlag{"192.0.2.1"}, name: "botnet", description: "override"},
			want: api.WAFListSet{
				Project: "acme", Name: "botnet", Description: "override", Type: api.WAFListTypeIP,
				Entries: []string{"203.0.113.0/24", "198.51.100.7", "192.0.2.1"},
			},
		},
		{
			name: "entries-file replaces spec entries, entry flags append",
			in:   wafListSetFlags{fn: spec, entriesFile: entriesFile, entries: multiFlag{"192.0.2.1"}},
			want: api.WAFListSet{
				Project: "acme", Name: "office-ips", Description: "office ranges", Type: api.WAFListTypeIP,
				Entries: []string{"192.0.2.0/24", "2001:db8::/48", "192.0.2.1"},
			},
		},
		{
			name: "entries-file with entry flags, no spec",
			in:   wafListSetFlags{project: "acme", name: "office-ips", entriesFile: entriesFile, entries: multiFlag{"192.0.2.1"}},
			want: api.WAFListSet{
				Project: "acme", Name: "office-ips",
				Entries: []string{"192.0.2.0/24", "2001:db8::/48", "192.0.2.1"},
			},
		},
		{
			name: "explicit empty entries in spec allowed",
			in:   wafListSetFlags{fn: emptySpec},
			want: api.WAFListSet{Project: "acme", Name: "office-ips", Entries: []string{}},
		},
		{
			name:    "no entries source",
			in:      wafListSetFlags{project: "acme", name: "office-ips"},
			wantErr: "entries required",
		},
		{
			name:    "spec without entries key",
			in:      wafListSetFlags{fn: noEntriesSpec},
			wantErr: "entries required",
		},
		{
			name:    "entries-file with no entries",
			in:      wafListSetFlags{project: "acme", name: "office-ips", entriesFile: blankEntriesFile},
			wantErr: "no entries",
		},
		{
			name:    "entries-file with no entries not masked by entry flags",
			in:      wafListSetFlags{project: "acme", name: "office-ips", entriesFile: blankEntriesFile, entries: multiFlag{"192.0.2.1"}},
			wantErr: "no entries",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildWAFListSet(tt.in)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("buildWAFListSet error = %v; want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildWAFListSet error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildWAFListSet = %+v; want %+v", got, tt.want)
			}
		})
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
