package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Port != 2222 {
		t.Errorf("expected port 2222, got %d", cfg.Port)
	}
	if cfg.PollInterval.Status != 10 {
		t.Errorf("expected status interval 10, got %d", cfg.PollInterval.Status)
	}
	if cfg.PollInterval.Agents != 5 {
		t.Errorf("expected agents interval 5, got %d", cfg.PollInterval.Agents)
	}
	if cfg.PollInterval.Convoys != 15 {
		t.Errorf("expected convoys interval 15, got %d", cfg.PollInterval.Convoys)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/kestral.yaml")
	if err != nil {
		t.Fatalf("missing file should return defaults, got error: %v", err)
	}
	if cfg.Port != 2222 {
		t.Errorf("expected default port 2222, got %d", cfg.Port)
	}
}

func TestLoadValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kestral.yaml")

	data := []byte(`port: 3333
town_root: /tmp/gt
host_key_dir: /tmp/keys
poll_interval:
  status: 20
  agents: 10
  convoys: 30
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 3333 {
		t.Errorf("expected port 3333, got %d", cfg.Port)
	}
	if cfg.TownRoot != "/tmp/gt" {
		t.Errorf("expected town_root /tmp/gt, got %s", cfg.TownRoot)
	}
	if cfg.HostKeyDir != "/tmp/keys" {
		t.Errorf("expected host_key_dir /tmp/keys, got %s", cfg.HostKeyDir)
	}
	if cfg.PollInterval.Status != 20 {
		t.Errorf("expected status 20, got %d", cfg.PollInterval.Status)
	}
	if cfg.PollInterval.Agents != 10 {
		t.Errorf("expected agents 10, got %d", cfg.PollInterval.Agents)
	}
	if cfg.PollInterval.Convoys != 30 {
		t.Errorf("expected convoys 30, got %d", cfg.PollInterval.Convoys)
	}
}

func TestLoadPartialYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kestral.yaml")

	data := []byte(`port: 4444
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 4444 {
		t.Errorf("expected port 4444, got %d", cfg.Port)
	}
	if cfg.PollInterval.Status != 10 {
		t.Errorf("expected default status 10, got %d", cfg.PollInterval.Status)
	}
}

func TestLoadInvalidPort(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kestral.yaml")

	data := []byte(`port: 99999
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error for invalid port")
	}
}

func TestLoadInvalidPollInterval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kestral.yaml")

	data := []byte(`poll_interval:
  status: 0
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error for zero poll interval")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kestral.yaml")

	data := []byte(`{{{not yaml`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error for invalid YAML")
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		got := expandPath(tt.input)
		if got != tt.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestLoadPortZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kestral.yaml")

	data := []byte(`port: 0
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error for port 0")
	}
}
