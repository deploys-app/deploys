package runner

import (
	"os"
	"testing"

	"github.com/deploys-app/api/client"
)

func TestFormatPublishProgress(t *testing.T) {
	cases := []struct {
		name string
		in   client.SitePublishProgress
		want string
	}{
		{
			name: "empty total is 0% and never divides by zero",
			in:   client.SitePublishProgress{Done: 0, Total: 0},
			want: "Uploading [------------------------]   0%  0/0 files  0 B/0 B",
		},
		{
			name: "halfway fills half the bar",
			in:   client.SitePublishProgress{Done: 5, Total: 10, BytesDone: 500, BytesTotal: 1000},
			want: "Uploading [############------------]  50%  5/10 files  500 B/1.0 kB",
		},
		{
			name: "complete fills the whole bar",
			in:   client.SitePublishProgress{Done: 30, Total: 30, BytesDone: 7800000, BytesTotal: 7800000, Uploaded: 22, Skipped: 8},
			want: "Uploading [########################] 100%  30/30 files  7.8 MB/7.8 MB",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatPublishProgress(c.in); got != c.want {
				t.Errorf("formatPublishProgress()\n got: %q\nwant: %q", got, c.want)
			}
		})
	}
}

func TestHumanByteSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{999, "999 B"},
		{1000, "1.0 kB"},
		{1500, "1.5 kB"},
		{7800000, "7.8 MB"},
		{3_200_000_000, "3.2 GB"},
	}
	for _, c := range cases {
		if got := humanByteSize(c.in); got != c.want {
			t.Errorf("humanByteSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A pipe is not a character device, so progress rendering must be suppressed and
// the callback/finish must be safe no-ops that write nothing.
func TestNewPublishProgressNonTerminalIsNoOp(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	if isTerminal(w) {
		t.Fatal("isTerminal(pipe) = true, want false")
	}

	progress, finish := newPublishProgress(w)
	progress(client.SitePublishProgress{Done: 1, Total: 2})
	finish()

	// Nothing should have been written to the pipe. Close the writer so a read
	// returns EOF immediately rather than blocking.
	w.Close()
	buf := make([]byte, 64)
	n, _ := r.Read(buf)
	if n != 0 {
		t.Errorf("non-terminal progress wrote %d bytes (%q), want 0", n, buf[:n])
	}
}
