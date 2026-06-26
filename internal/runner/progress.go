package runner

import (
	"fmt"
	"os"
	"strings"

	"github.com/deploys-app/api/client"
)

// newPublishProgress returns a progress callback to hand to
// client.SitePublishOptions.Progress, together with a finish func to call once
// the publish returns. It renders a single-line, in-place upload progress bar
// to w when w is a terminal; when w is not a terminal (a pipe, a file, CI) both
// the callback and finish are no-ops, so redirected output and CI logs stay
// clean. Callers draw on stderr so the bar never interleaves with -output
// json/yaml on stdout.
func newPublishProgress(w *os.File) (progress func(client.SitePublishProgress), finish func()) {
	if !isTerminal(w) {
		return func(client.SitePublishProgress) {}, func() {}
	}

	var (
		lastLen int
		started bool
	)
	progress = func(p client.SitePublishProgress) {
		started = true
		line := formatPublishProgress(p)
		// Pad with spaces to erase any remnant of a previously longer line.
		pad := ""
		if d := lastLen - len(line); d > 0 {
			pad = strings.Repeat(" ", d)
		}
		fmt.Fprintf(w, "\r%s%s", line, pad)
		lastLen = len(line)
	}
	finish = func() {
		if started {
			fmt.Fprintln(w) // terminate the in-place line with a newline
		}
	}
	return progress, finish
}

// formatPublishProgress renders one progress line (no carriage return, no
// padding). Kept pure and ASCII-only so it can be unit-tested and never
// mis-measures column width on a redraw.
func formatPublishProgress(p client.SitePublishProgress) string {
	const barWidth = 24

	pct, filled := 0, 0
	if p.Total > 0 {
		pct = p.Done * 100 / p.Total
		filled = p.Done * barWidth / p.Total
	}
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)

	return fmt.Sprintf("Uploading [%s] %3d%%  %d/%d files  %s/%s",
		bar, pct, p.Done, p.Total,
		humanByteSize(p.BytesDone), humanByteSize(p.BytesTotal))
}

// isTerminal reports whether f is a character device (a terminal), so progress
// rendering is skipped for pipes, files and CI. A char-device mode bit is a
// good-enough "interactive" heuristic on the platforms the CLI targets and
// avoids pulling in a terminal dependency.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// humanByteSize renders a byte count compactly (e.g. "3.2 MB"). Base-10 units
// match what users expect when reading upload sizes.
func humanByteSize(n int64) string {
	const unit = 1000
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "kMGTPE"[exp])
}
