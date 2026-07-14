package runner

import (
	"reflect"
	"testing"

	"github.com/deploys-app/api"
	"gopkg.in/yaml.v2"
)

// TestWAFSetSpecManagedRules pins the spec-file contract the waf set help
// promises (SPEC-waf-managed-rules §2.2/§6): the CLI never touches
// managedRules itself — it relies on the api module's yaml tags — so an api
// tag regression would silently drop or rename the field on the wire. These
// cases fail loudly instead.
func TestWAFSetSpecManagedRules(t *testing.T) {
	t.Parallel()

	parse := func(t *testing.T, spec string) api.WAFSet {
		t.Helper()
		var req api.WAFSet
		if err := yaml.Unmarshal([]byte(spec), &req); err != nil {
			t.Fatalf("parse spec: %v", err)
		}
		return req
	}

	t.Run("OmittedDecodesNil", func(t *testing.T) {
		t.Parallel()

		// A pre-managedRules spec must decode to nil (= server disables and
		// clears the block under whole-replace semantics), not a zero struct.
		req := parse(t, "project: p1\nlocation: gke.cluster-rcf2\nrules: []\nlimits: []\n")
		if req.ManagedRules != nil {
			t.Fatalf("managedRules omitted, want nil, got %+v", req.ManagedRules)
		}
	})

	t.Run("DisabledKeepsTuning", func(t *testing.T) {
		t.Parallel()

		// enabled:false with tuning is the mid-incident toggle-off shape; the
		// curated exclusion list must survive the CLI parse.
		req := parse(t, `
project: p1
location: gke.cluster-rcf2
managedRules:
  enabled: false
  excludedRules: [942100]
`)
		want := &api.WAFManagedRules{Enabled: false, ExcludedRules: []int{942100}}
		if !reflect.DeepEqual(req.ManagedRules, want) {
			t.Fatalf("managedRules = %+v, want %+v", req.ManagedRules, want)
		}
	})

	t.Run("GetOutputRoundTrips", func(t *testing.T) {
		t.Parallel()

		// The documented workflow is `waf get -oyaml` → edit → `waf set -f`;
		// WAFItem yaml (with its read-only extras) must feed back into WAFSet
		// with the full managedRules block intact.
		mr := &api.WAFManagedRules{
			Enabled:          true,
			Mode:             "detect",
			ParanoiaLevel:    2,
			AnomalyThreshold: 10,
			ExcludedRules:    []int{941100, 942100},
		}
		item := api.WAFItem{
			Project:      "p1",
			Location:     "gke.cluster-rcf2",
			Description:  "zone",
			ManagedRules: mr,
			Status:       api.Success,
		}
		b, err := yaml.Marshal(item)
		if err != nil {
			t.Fatalf("marshal waf get output: %v", err)
		}
		req := parse(t, string(b))
		if !reflect.DeepEqual(req.ManagedRules, mr) {
			t.Fatalf("round-trip managedRules = %+v, want %+v", req.ManagedRules, mr)
		}
	})
}
