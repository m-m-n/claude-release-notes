// Package config loads and validates the tool's YAML configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the validated configuration values required to run the tool.
type Config struct {
	GmailAccount     string
	GmailAppPassword string
	MailTo           string
	GitHubToken      string
}

// fileFormat mirrors the on-disk YAML structure (SPEC.md "Database Schema").
type fileFormat struct {
	Gmail struct {
		Account     string `yaml:"account"`
		AppPassword string `yaml:"app_password"`
	} `yaml:"gmail"`
	Mail struct {
		To string `yaml:"to"`
	} `yaml:"mail"`
	GitHub struct {
		Token string `yaml:"token"`
	} `yaml:"github"`
}

// appDirName is the XDG application directory name under the config home.
const appDirName = "claude-release-notes"

// Load reads and validates config.yaml from baseDir. When baseDir is empty,
// the XDG config home is resolved internally (XDG_CONFIG_HOME, falling back
// to ~/.config when unset or empty) with "claude-release-notes" appended.
//
// On success every field of the returned Config is non-empty. On failure the
// error names the file path or the offending config key; it never contains a
// configured secret value.
func Load(baseDir string) (Config, error) {
	dir := baseDir
	if dir == "" {
		var err error
		dir, err = defaultConfigDir()
		if err != nil {
			return Config{}, fmt.Errorf("config: %w", err)
		}
	}

	path := filepath.Join(dir, "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	var ff fileFormat
	if err := yaml.Unmarshal(data, &ff); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}

	cfg := Config{
		GmailAccount:     ff.Gmail.Account,
		GmailAppPassword: ff.Gmail.AppPassword,
		MailTo:           ff.Mail.To,
		GitHubToken:      ff.GitHub.Token,
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// validate checks that every required key is present and non-empty. The
// returned error names the first offending key; it never includes the
// key's value.
func validate(cfg Config) error {
	fields := []struct {
		key   string
		value string
	}{
		{"gmail.account", cfg.GmailAccount},
		{"gmail.app_password", cfg.GmailAppPassword},
		{"mail.to", cfg.MailTo},
		{"github.token", cfg.GitHubToken},
	}
	for _, f := range fields {
		if f.value == "" {
			return fmt.Errorf("config: missing required key %s", f.key)
		}
	}
	return nil
}

// defaultConfigDir resolves the XDG config directory for this tool:
// $XDG_CONFIG_HOME/claude-release-notes, falling back to
// ~/.config/claude-release-notes when XDG_CONFIG_HOME is unset or empty.
func defaultConfigDir() (string, error) {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, appDirName), nil
}
