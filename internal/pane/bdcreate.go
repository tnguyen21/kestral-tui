package pane

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const bdTimeout = 15 * time.Second

// runBdCreate executes bd create with the given args and returns stdout.
func runBdCreate(args []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), bdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bd", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("bd timed out after %v", bdTimeout)
		}
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("bd create: %s", errMsg)
		}
		return "", fmt.Errorf("bd create: %w", err)
	}
	return stdout.String(), nil
}

// beadIDPattern matches common bead ID formats (e.g., kt-abc1, gt-xyz9).
var beadIDPattern = regexp.MustCompile(`[a-z]{2,}-[a-z0-9]{3,}`)

// parseBeadID extracts a bead ID from bd create output.
func parseBeadID(output string) string {
	// Try to find a bead ID pattern in the output
	match := beadIDPattern.FindString(output)
	if match != "" {
		return match
	}
	// Fallback: return first non-empty line trimmed
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "(unknown)"
}
