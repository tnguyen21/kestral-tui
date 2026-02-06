package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "~/.config/kestral/kestral.yaml"

type PollInterval struct {
	Status  int `yaml:"status"`
	Agents  int `yaml:"agents"`
	Convoys int `yaml:"convoys"`
}

type Config struct {
	Port         int          `yaml:"port"`
	TownRoot     string       `yaml:"town_root"`
	HostKeyDir   string       `yaml:"host_key_dir"`
	PollInterval PollInterval `yaml:"poll_interval"`
}

func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Port:       2222,
		TownRoot:   filepath.Join(home, "gt"),
		HostKeyDir: filepath.Join(home, ".ssh"),
		PollInterval: PollInterval{
			Status:  10,
			Agents:  5,
			Convoys: 15,
		},
	}
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func Load(path string) (Config, error) {
	cfg := Default()

	resolved := expandPath(path)
	data, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config %s: %w", resolved, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", resolved, err)
	}

	cfg.TownRoot = expandPath(cfg.TownRoot)
	cfg.HostKeyDir = expandPath(cfg.HostKeyDir)

	if err := validate(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func validate(cfg Config) error {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("port %d out of range (1-65535)", cfg.Port)
	}

	if cfg.PollInterval.Status < 1 {
		return fmt.Errorf("poll_interval.status must be >= 1")
	}
	if cfg.PollInterval.Agents < 1 {
		return fmt.Errorf("poll_interval.agents must be >= 1")
	}
	if cfg.PollInterval.Convoys < 1 {
		return fmt.Errorf("poll_interval.convoys must be >= 1")
	}

	return nil
}
