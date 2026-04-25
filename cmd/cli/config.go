package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func writeMergedConfig(updates map[string]any) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".casper")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.json")

	cfg := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &cfg)
	}
	for k, v := range updates {
		if s, ok := v.(string); ok && s == "" {
			delete(cfg, k)
			continue
		}
		cfg[k] = v
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	_ = viper.ReadInConfig()
	return nil
}
