package runner

import (
	"strings"
	"testing"
)

func TestMetricSourceHelp(t *testing.T) {
	c := lookupCommand("metricsource")
	if c == nil {
		t.Fatal("metricsource group missing")
	}
	for _, name := range []string{"set", "get", "list", "delete", "series", "query"} {
		if c.lookupSub(name) == nil {
			t.Errorf("metricsource missing subcommand %q", name)
		}
	}
	query := c.lookupSub("query")
	if query == nil {
		t.Fatal("metricsource query missing")
	}
	if !strings.Contains(query.args, "timerange") {
		t.Errorf("query args %q missing timerange", query.args)
	}
	set := c.lookupSub("set")
	if set == nil {
		t.Fatal("metricsource set missing")
	}
	for _, want := range []string{"location", "deployment", "port", "path"} {
		if !strings.Contains(set.args, want) {
			t.Errorf("set args %q missing %q", set.args, want)
		}
	}
}
