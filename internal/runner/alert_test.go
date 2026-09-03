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

func TestAlertCreateHelpMentionsKindAndSource(t *testing.T) {
	c := lookupCommand("alert")
	if c == nil {
		t.Fatal("alert group missing")
	}
	sub := c.lookupSub("create")
	if sub == nil {
		t.Fatal("alert create missing")
	}
	for _, want := range []string{"kind", "source", "value", "rate"} {
		if !strings.Contains(sub.args, want) {
			t.Errorf("create args %q missing %q", sub.args, want)
		}
	}
}

func TestApplyAlertTargetKind(t *testing.T) {
	custom := api.AlertTarget{
		Kind:       api.AlertTargetKindCustom,
		Location:   "gke.cluster-rcf2",
		Deployment: "web",
		Source:     "app",
		Series:     `queue_depth{queue="email"}`,
	}
	applyAlertTargetKind(&custom)
	if custom.Kind != api.AlertTargetKindCustom {
		t.Errorf("custom kind = %q; want %q", custom.Kind, api.AlertTargetKindCustom)
	}
	if custom.Location != "" || custom.Deployment != "" {
		t.Errorf("custom target kept deployment fields: %+v", custom)
	}
	if custom.Source != "app" || custom.Series != `queue_depth{queue="email"}` {
		t.Errorf("custom target dropped source/series: %+v", custom)
	}

	dep := api.AlertTarget{
		Kind:       "",
		Location:   "gke.cluster-rcf2",
		Deployment: "web",
		Source:     "app",
		Series:     `queue_depth{queue="email"}`,
	}
	applyAlertTargetKind(&dep)
	if dep.Kind != "" {
		t.Errorf("empty kind rewritten to %q; want empty", dep.Kind)
	}
	if dep.Location != "gke.cluster-rcf2" || dep.Deployment != "web" {
		t.Errorf("deployment target dropped location/deployment: %+v", dep)
	}
	if dep.Source != "" || dep.Series != "" {
		t.Errorf("deployment target kept custom fields: %+v", dep)
	}
}
