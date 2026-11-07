package engine

import (
	"strings"
	"testing"
)

func TestTailWriterKeepsEnd(t *testing.T) {
	w := &tailWriter{}
	_, _ = w.Write([]byte(strings.Repeat("a", daemonTailBytes)))
	_, _ = w.Write([]byte("the error"))
	got := w.String()
	if len(got) != daemonTailBytes || !strings.HasSuffix(got, "the error") {
		t.Fatalf("len %d, suffix %q", len(got), got[len(got)-12:])
	}
}
