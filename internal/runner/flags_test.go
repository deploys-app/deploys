package runner

import (
	"testing"
	"time"
)

func TestParseKV(t *testing.T) {
	m, err := parseKV(nil)
	if err != nil || m != nil {
		t.Errorf("parseKV(nil) = %v, %v; want nil, nil", m, err)
	}

	m, err = parseKV([]string{"A=1", "B=x=y", "C="})
	if err != nil {
		t.Fatalf("parseKV() error: %v", err)
	}
	if m["A"] != "1" || m["B"] != "x=y" || m["C"] != "" {
		t.Errorf("parseKV() = %v", m)
	}

	for _, kv := range []string{"A", "=1"} {
		_, err = parseKV([]string{kv})
		if err == nil {
			t.Errorf("parseKV(%q) want error", kv)
		}
	}
}

func TestTimeFlag(t *testing.T) {
	var v time.Time
	f := timeFlag{&v}

	err := f.Set("2026-06-12")
	if err != nil {
		t.Fatalf("Set(date) error: %v", err)
	}
	if v.Format("2006-01-02") != "2026-06-12" {
		t.Errorf("Set(date) = %v", v)
	}

	err = f.Set("2026-06-12T10:30:00Z")
	if err != nil {
		t.Fatalf("Set(rfc3339) error: %v", err)
	}
	if !v.Equal(time.Date(2026, 6, 12, 10, 30, 0, 0, time.UTC)) {
		t.Errorf("Set(rfc3339) = %v", v)
	}

	err = f.Set("yesterday")
	if err == nil {
		t.Error("Set(invalid) want error")
	}
}

func TestSplitComma(t *testing.T) {
	if got := splitComma(""); got != nil {
		t.Errorf("splitComma(\"\") = %v; want nil", got)
	}
	got := splitComma("a, b,,c")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("splitComma() = %v", got)
	}
}
