package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCheckProgressThrottlesAndClears(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	progress := checkProgress{writer: &output}
	progress.update(1, 10)
	if got := output.String(); got != "\r\x1b[2KChecking 1/10" {
		t.Fatalf("first update = %q", got)
	}
	// A future timestamp keeps the throttle test independent of scheduler delays.
	progress.last = time.Now().Add(time.Minute)
	progress.update(2, 10)
	if strings.Contains(output.String(), "2/10") {
		t.Fatalf("unthrottled update = %q", output.String())
	}
	progress.last = time.Time{}
	progress.update(3, 10)
	if !strings.Contains(output.String(), "Checking 3/10") {
		t.Fatalf("next update = %q", output.String())
	}
	progress.update(10, 10)
	if progress.visible || !strings.HasSuffix(output.String(), "\r\x1b[2K") {
		t.Fatalf("completion did not clear progress: %q", output.String())
	}
	length := output.Len()
	progress.clear()
	if output.Len() != length {
		t.Fatalf("clear wrote after completion: %q", output.String())
	}
}

func TestCheckProgressClearsOnError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	progress := checkProgress{writer: &output}
	progress.update(1, 3)
	// runLoaded clears the active line immediately after Evaluate returns an error.
	progress.clear()
	if progress.visible || !strings.HasSuffix(output.String(), "\r\x1b[2K") {
		t.Fatalf("error cleanup did not clear progress: %q", output.String())
	}
}

func TestCheckProgressTerminalDetectionRejectsBuffers(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	if isTerminal(&output) {
		t.Fatal("redirected output treated as a terminal")
	}
}
