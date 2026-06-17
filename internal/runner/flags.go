package runner

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

// visitedFlags returns the set of flag names that were explicitly provided on
// the command line. It lets a command distinguish "flag omitted" (leave the
// request field nil so the server keeps the previous value) from "flag set to
// the zero value" (e.g. -ttl 0 to clear, -internal=false to set false).
func visitedFlags(f *flag.FlagSet) map[string]bool {
	set := map[string]bool{}
	f.Visit(func(fl *flag.Flag) { set[fl.Name] = true })
	return set
}

// multiFlag collects a repeatable string flag (e.g. -env A=1 -env B=2).
type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// parseKV parses KEY=VALUE pairs into a map. Returns nil on empty input so
// untouched request fields stay nil.
func parseKV(kvs []string) (map[string]string, error) {
	if len(kvs) == 0 {
		return nil, nil
	}
	res := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid KEY=VALUE pair: '%s'", kv)
		}
		res[k] = v
	}
	return res, nil
}

// timeFlag parses an RFC 3339 timestamp or a date (2006-01-02) into time.Time.
type timeFlag struct {
	t *time.Time
}

func (f timeFlag) String() string {
	if f.t == nil || f.t.IsZero() {
		return ""
	}
	return f.t.Format(time.RFC3339)
}

func (f timeFlag) Set(v string) error {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		t, err := time.Parse(layout, v)
		if err == nil {
			*f.t = t
			return nil
		}
	}
	return fmt.Errorf("invalid time '%s' (want RFC 3339 or YYYY-MM-DD)", v)
}

// splitComma splits a comma separated value, dropping empty elements.
// Returns nil on empty input.
func splitComma(s string) []string {
	var res []string
	for x := range strings.SplitSeq(s, ",") {
		x = strings.TrimSpace(x)
		if x != "" {
			res = append(res, x)
		}
	}
	return res
}
