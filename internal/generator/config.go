package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds per-app generator defaults, read from .gor/config.json —
// the gor equivalent of Rails' config.generators. Explicit CLI flags
// always override these.
//
//	{ "resource": { "headless": true, "service": true } }
type Config struct {
	Resource ResourceDefaults `json:"resource"`
}

// ResourceDefaults are the default flags for `gor g resource`.
type ResourceDefaults struct {
	Headless  bool `json:"headless"`
	Service   bool `json:"service"`
	APIOnly   bool `json:"apiOnly"`
	Protected bool `json:"protected"`
	Live      bool `json:"live"`
}

const configPath = ".gor/config.json"

// LoadConfig reads the app's generator defaults; a missing file means
// zero-value defaults.
func LoadConfig(root string) (Config, error) {
	raw, err := os.ReadFile(filepath.Join(root, configPath))
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", configPath, err)
	}
	return cfg, nil
}
