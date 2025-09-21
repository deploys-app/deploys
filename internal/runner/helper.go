package runner

import (
	"fmt"
	"os"
	"strings"
)

func splitCommaList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// format: "K=V,A=B" (values may be empty). No escaping; keep it simple like existing flags.
func parseKVList(s string) map[string]string {
	m := map[string]string{}
	for _, pair := range splitCommaList(s) {
		if eq := strings.IndexByte(pair, '='); eq >= 0 {
			k := strings.TrimSpace(pair[:eq])
			v := strings.TrimSpace(pair[eq+1:])
			if k != "" {
				m[k] = v
			}
		} else if pair != "" {
			// allow "KEY" -> empty value
			m[pair] = ""
		}
	}
	return m
}

// supports "@path" to read file content; otherwise returns value as-is
func readMaybeFile(value string) (string, error) {
	if strings.HasPrefix(value, "@") && len(value) > 1 {
		b, err := os.ReadFile(value[1:])
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return value, nil
}

// format: "/path=VALUE,/path2=@file.txt"
func parseMountData(s string) (map[string]string, error) {
	m := map[string]string{}
	for _, pair := range splitCommaList(s) {
		if eq := strings.IndexByte(pair, '='); eq >= 0 {
			k := strings.TrimSpace(pair[:eq])
			v := strings.TrimSpace(pair[eq+1:])
			if k == "" || !strings.HasPrefix(k, "/") {
				return nil, fmt.Errorf("mountData key must be absolute path: %q", k)
			}
			val, err := readMaybeFile(v)
			if err != nil {
				return nil, err
			}
			m[k] = val
		} else if pair != "" {
			return nil, fmt.Errorf("mountData must be key=value: %q", pair)
		}
	}
	return m, nil
}
