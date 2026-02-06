package data

import (
	"testing"
	"time"
)

func TestTimeoutConstants(t *testing.T) {
	if cmdTimeout != 15*time.Second {
		t.Errorf("cmdTimeout = %v, want 15s", cmdTimeout)
	}
	if ghCmdTimeout != 10*time.Second {
		t.Errorf("ghCmdTimeout = %v, want 10s", ghCmdTimeout)
	}
	if tmuxCmdTimeout != 2*time.Second {
		t.Errorf("tmuxCmdTimeout = %v, want 2s", tmuxCmdTimeout)
	}
}

func TestRunCmdMissingBinary(t *testing.T) {
	_, err := runCmd(2*time.Second, "nonexistent-binary-xyz")
	if err == nil {
		t.Error("expected error for missing binary, got nil")
	}
}

func TestFetcherRunBdCmdMissingBinary(t *testing.T) {
	f := &Fetcher{TownRoot: t.TempDir()}
	_, err := f.runBdCmd("list")
	if err == nil {
		t.Error("expected error when bd is not on PATH, got nil")
	}
}

func TestRunCmdTimeout(t *testing.T) {
	_, err := runCmd(100*time.Millisecond, "sleep", "10")
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
	if err != nil && !contains(err.Error(), "timed out") {
		t.Errorf("expected timeout message, got: %v", err)
	}
}

func TestRunCmdSuccess(t *testing.T) {
	buf, err := runCmd(2*time.Second, "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := buf.String(); got != "hello\n" {
		t.Errorf("got %q, want %q", got, "hello\n")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsAt(s, sub))
}

func containsAt(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
