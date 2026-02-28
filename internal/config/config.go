package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
)

// RemoteConfig holds SSH connection settings for remote sync.
type RemoteConfig struct {
	Host string
	User string
	Port int
	Key  string
	Path string
}

type tomlFile struct {
	Remote struct {
		Host string `toml:"host"`
		User string `toml:"user"`
		Port int    `toml:"port"`
		Key  string `toml:"key"`
		Path string `toml:"path"`
	} `toml:"remote"`
}

// Load reads RemoteConfig with precedence: env vars > ~/.config/notizen/config.toml.
func Load() (*RemoteConfig, error) {
	cfg, err := loadFromFile()
	if err != nil {
		return nil, err
	}
	if err := applyEnv(cfg); err != nil {
		return nil, err
	}
	return validate(cfg)
}

func loadFromFile() (*RemoteConfig, error) {
	configPath, err := defaultConfigPath()
	if err != nil {
		return nil, err
	}
	var f tomlFile
	if _, err := toml.DecodeFile(configPath, &f); err != nil {
		// Missing or unreadable config file is not fatal; env vars may provide all values.
		return &RemoteConfig{}, nil
	}
	return &RemoteConfig{
		Host: f.Remote.Host,
		User: f.Remote.User,
		Port: f.Remote.Port,
		Key:  f.Remote.Key,
		Path: f.Remote.Path,
	}, nil
}

func applyEnv(cfg *RemoteConfig) error {
	if v := os.Getenv("NOTIZEN_REMOTE_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("NOTIZEN_REMOTE_USER"); v != "" {
		cfg.User = v
	}
	if v := os.Getenv("NOTIZEN_REMOTE_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid NOTIZEN_REMOTE_PORT %q: %w", v, err)
		}
		cfg.Port = p
	}
	if v := os.Getenv("NOTIZEN_REMOTE_KEY"); v != "" {
		cfg.Key = v
	}
	if v := os.Getenv("NOTIZEN_REMOTE_PATH"); v != "" {
		cfg.Path = v
	}
	return nil
}

func validate(cfg *RemoteConfig) (*RemoteConfig, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("remote host is not set (use NOTIZEN_REMOTE_HOST or [remote] host in config.toml)")
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d (must be 1-65535)", cfg.Port)
	}
	if cfg.Key != "" {
		if _, err := os.Stat(cfg.Key); err != nil {
			return nil, fmt.Errorf("SSH key file not found: %s", cfg.Key)
		}
	}
	return cfg, nil
}

func defaultConfigPath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "notizen", "config.toml"), nil
}
