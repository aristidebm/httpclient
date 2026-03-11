package session

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultEnv    string `yaml:"default_env"`
	DefaultEditor string `yaml:"default_editor"`
	HistoryFile   string `yaml:"history_file"`
}

func LoadConfig() (*Config, error) {
	path, err := expandPath(ConfigFile)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				DefaultEnv:    "local",
				DefaultEditor: "",
				HistoryFile:   "~/.httpclient/history",
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.HistoryFile == "" {
		cfg.HistoryFile = "~/.httpclient/history"
	}

	return &cfg, nil
}

func SaveConfig(c *Config) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}

	path, err := expandPath(ConfigFile)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
