package runner

import (
	"strings"
	"testing"

	"github.com/deploys-app/api"
)

func TestNormalizeAlertOp(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", api.AlertOpGTE},
		{"  ", api.AlertOpGTE},
		{">=", api.AlertOpGTE},
		{"gte", api.AlertOpGTE},
		{"GTE", api.AlertOpGTE},
		{"<=", api.AlertOpLTE},
		{"lte", api.AlertOpLTE},
		{"LTE", api.AlertOpLTE},
		{"gt", "gt"}, // unknown values pass through for API Valid() to reject
	}
	for _, tc := range cases {
		if got := normalizeAlertOp(tc.in); got != tc.want {
			t.Errorf("normalizeAlertOp(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestAlertCreateHelpMentionsShellSafeOp(t *testing.T) {
	c := lookupCommand("alert")
	if c == nil {
		t.Fatal("alert group missing")
	}
	sub := c.lookupSub("create")
	if sub == nil {
		t.Fatal("alert create missing")
	}
	for _, want := range []string{"gte", "lte"} {
		if !strings.Contains(sub.args, want) {
			t.Errorf("create args %q missing shell-safe op alias %q", sub.args, want)
		}
	}
}
