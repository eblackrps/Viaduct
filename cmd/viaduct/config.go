package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eblackrps/viaduct/internal/connectors"
	"gopkg.in/yaml.v3"
)

type appConfig struct {
	Username      string                       `yaml:"username"`
	Password      string                       `yaml:"password"`
	Insecure      bool                         `yaml:"insecure"`
	StateStoreDSN string                       `yaml:"state_store_dsn"`
	Sources       map[string]connectors.Config `yaml:"sources"`
	Plugins       map[string]string            `yaml:"plugins"`
}

func loadAppConfig(path string) (*appConfig, error) {
	expandedPath, err := expandPath(path)
	if err != nil {
		return nil, fmt.Errorf("load config path: %w", err)
	}

	payload, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &appConfig{}, nil
		}

		return nil, fmt.Errorf("read config %s: %w", expandedPath, err)
	}

	var cfg appConfig
	if err := yaml.Unmarshal(payload, &cfg); err != nil {
		return nil, fmt.Errorf("decode config %s: %w", expandedPath, err)
	}

	return &cfg, nil
}

func resolveConnectorConfig(source, platform, flagUsername, flagPassword string, insecure bool, cfg *appConfig) connectors.Config {
	config := connectors.Config{
		Address:  source,
		Username: strings.TrimSpace(flagUsername),
		Password: strings.TrimSpace(flagPassword),
		Insecure: insecure,
	}

	if cfg == nil {
		cfg = &appConfig{}
	}

	if sourceConfig, ok := cfg.Sources[source]; ok {
		mergeConnectorConfig(&config, sourceConfig)
	} else if platformConfig, ok := cfg.Sources[platform]; ok {
		mergeConnectorConfig(&config, platformConfig)
	}

	if envUsername := strings.TrimSpace(os.Getenv("VIADUCT_USERNAME")); envUsername != "" {
		config.Username = envUsername
	}
	if envPassword := strings.TrimSpace(os.Getenv("VIADUCT_PASSWORD")); envPassword != "" {
		config.Password = envPassword
	}

	if cfg.Username != "" && config.Username == "" {
		config.Username = cfg.Username
	}
	if cfg.Password != "" && config.Password == "" {
		config.Password = cfg.Password
	}
	if !config.Insecure && cfg.Insecure {
		config.Insecure = true
	}

	if strings.TrimSpace(flagUsername) != "" {
		config.Username = strings.TrimSpace(flagUsername)
	}
	if strings.TrimSpace(flagPassword) != "" {
		config.Password = strings.TrimSpace(flagPassword)
	}
	if insecure {
		config.Insecure = true
	}

	return config
}

func mergeConnectorConfig(target *connectors.Config, source connectors.Config) {
	if source.Address != "" {
		target.Address = source.Address
	}
	if target.Username == "" && source.Username != "" {
		target.Username = source.Username
	}
	if target.Password == "" && source.Password != "" {
		target.Password = source.Password
	}
	if !target.Insecure && source.Insecure {
		target.Insecure = true
	}
	if target.Port == 0 && source.Port != 0 {
		target.Port = source.Port
	}
}

func expandPath(path string) (string, error) {
	if path == "" {
		return path, nil
	}

	if strings.HasPrefix(path, "~\\") || strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}

		return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~\\"), "~/")), nil
	}

	return path, nil
}
