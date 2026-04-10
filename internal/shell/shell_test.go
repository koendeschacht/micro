package shell

import (
	"strings"
	"testing"
	"time"
)

func TestExecCommandWithTimeout(t *testing.T) {
	_, err := execCommandWithTimeout(50*time.Millisecond, "sh", "-c", "sleep 1")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "command timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestExecCommandWithTimeoutCapturesOutput(t *testing.T) {
	output, err := execCommandWithTimeout(1*time.Second, "sh", "-c", "printf hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "hello" {
		t.Fatalf("expected output hello, got %q", output)
	}
}
