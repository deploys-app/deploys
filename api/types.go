package api

import (
	"fmt"
	"time"
)

type Empty struct{}

func (*Empty) Table() [][]string {
	return [][]string{{"Operation success"}}
}

func Int(i int) *int {
	return &i
}

func Int64(i int64) *int64 {
	return &i
}

func String(s string) *string {
	return &s
}

func Bool(b bool) *bool {
	return &b
}

func age(t time.Time) string {
	d := time.Since(t)
	if x := d / (24 * time.Hour); x > 0 {
		return fmt.Sprintf("%dd", x)
	}
	if x := d / (24 * time.Hour); x > 0 {
		return fmt.Sprintf("%dh", x)
	}
	if x := d / time.Minute; x > 0 {
		return fmt.Sprintf("%dm", x)
	}
	return fmt.Sprintf("%ds", d/time.Second)
}
