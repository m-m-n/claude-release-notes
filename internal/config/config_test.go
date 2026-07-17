package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig writes content to <dir>/config.yaml, creating dir as needed.
func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestLoad_ValidConfigParsesAllFields references AC-1.
func TestLoad_ValidConfigParsesAllFields(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
gmail:
  account: user@example.com
  app_password: "abcd efgh ijkl mnop"
mail:
  to: recipient@example.com
github:
  token: ghp_exampletoken
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	want := Config{
		GmailAccount:     "user@example.com",
		GmailAppPassword: "abcd efgh ijkl mnop",
		MailTo:           "recipient@example.com",
		GitHubToken:      "ghp_exampletoken",
	}
	if cfg != want {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

// TestLoad_MissingOrEmptyRequiredKey references AC-2 (error names the missing
// key, for both an absent key and a present-but-empty key) and AC-3 (the
// error never contains a configured secret value, asserted via sentinels).
func TestLoad_MissingOrEmptyRequiredKey(t *testing.T) {
	const sentinelAppPassword = "SENTINEL-APP-PASSWORD-af38x"
	const sentinelToken = "SENTINEL-GITHUB-TOKEN-91kd"

	tests := []struct {
		name    string
		yaml    string
		wantKey string
	}{
		{
			name: "gmail.account missing",
			yaml: `
gmail:
  app_password: "` + sentinelAppPassword + `"
mail:
  to: recipient@example.com
github:
  token: ` + sentinelToken + `
`,
			wantKey: "gmail.account",
		},
		{
			name: "gmail.account empty",
			yaml: `
gmail:
  account: ""
  app_password: "` + sentinelAppPassword + `"
mail:
  to: recipient@example.com
github:
  token: ` + sentinelToken + `
`,
			wantKey: "gmail.account",
		},
		{
			name: "gmail.app_password missing",
			yaml: `
gmail:
  account: user@example.com
mail:
  to: recipient@example.com
github:
  token: ` + sentinelToken + `
`,
			wantKey: "gmail.app_password",
		},
		{
			name: "gmail.app_password empty",
			yaml: `
gmail:
  account: user@example.com
  app_password: ""
mail:
  to: recipient@example.com
github:
  token: ` + sentinelToken + `
`,
			wantKey: "gmail.app_password",
		},
		{
			name: "mail.to missing",
			yaml: `
gmail:
  account: user@example.com
  app_password: "` + sentinelAppPassword + `"
github:
  token: ` + sentinelToken + `
`,
			wantKey: "mail.to",
		},
		{
			name: "mail.to empty",
			yaml: `
gmail:
  account: user@example.com
  app_password: "` + sentinelAppPassword + `"
mail:
  to: ""
github:
  token: ` + sentinelToken + `
`,
			wantKey: "mail.to",
		},
		{
			name: "github.token missing",
			yaml: `
gmail:
  account: user@example.com
  app_password: "` + sentinelAppPassword + `"
mail:
  to: recipient@example.com
`,
			wantKey: "github.token",
		},
		{
			name: "github.token empty",
			yaml: `
gmail:
  account: user@example.com
  app_password: "` + sentinelAppPassword + `"
mail:
  to: recipient@example.com
github:
  token: ""
`,
			wantKey: "github.token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, tt.yaml)

			_, err := Load(dir)
			if err == nil {
				t.Fatalf("Load() returned nil error, want error naming %q", tt.wantKey)
			}
			if !strings.Contains(err.Error(), tt.wantKey) {
				t.Errorf("Load() error = %q, want it to contain %q", err.Error(), tt.wantKey)
			}
			if strings.Contains(err.Error(), sentinelAppPassword) {
				t.Errorf("Load() error = %q, must not contain the app_password value", err.Error())
			}
			if strings.Contains(err.Error(), sentinelToken) {
				t.Errorf("Load() error = %q, must not contain the github.token value", err.Error())
			}
		})
	}
}

// TestLoad_NonEmptyBaseDirIgnoresEnvironment references AC-4: with a
// non-empty baseDir, no environment variable or real home directory is
// consulted. XDG_CONFIG_HOME and HOME are pointed at directories with no
// valid config so the test fails loudly if Load falls back to them instead
// of exclusively reading baseDir.
func TestLoad_NonEmptyBaseDirIgnoresEnvironment(t *testing.T) {
	bogusXDG := t.TempDir()
	bogusHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", bogusXDG)
	t.Setenv("HOME", bogusHome)

	dir := t.TempDir()
	writeConfig(t, dir, `
gmail:
  account: user@example.com
  app_password: "app-password-value"
mail:
  to: recipient@example.com
github:
  token: token-value
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.GmailAccount != "user@example.com" {
		t.Errorf("Load() read from an unexpected source: got GmailAccount=%q", cfg.GmailAccount)
	}
}

// TestLoad_XDGResolution references AC-5.
func TestLoad_XDGResolution(t *testing.T) {
	t.Run("XDG_CONFIG_HOME set", func(t *testing.T) {
		xdgHome := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdgHome)
		// HOME must not matter in this branch; point it somewhere invalid.
		t.Setenv("HOME", t.TempDir())

		configDir := filepath.Join(xdgHome, "claude-release-notes")
		writeConfig(t, configDir, `
gmail:
  account: xdg-user@example.com
  app_password: "xdg-app-password"
mail:
  to: xdg-recipient@example.com
github:
  token: xdg-token
`)

		cfg, err := Load("")
		if err != nil {
			t.Fatalf(`Load("") returned error: %v`, err)
		}
		if cfg.GmailAccount != "xdg-user@example.com" {
			t.Errorf(`Load("") did not read from XDG_CONFIG_HOME: got GmailAccount=%q`, cfg.GmailAccount)
		}
	})

	t.Run("XDG_CONFIG_HOME unset falls back to home directory default", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		home := t.TempDir()
		t.Setenv("HOME", home)

		configDir := filepath.Join(home, ".config", "claude-release-notes")
		writeConfig(t, configDir, `
gmail:
  account: home-user@example.com
  app_password: "home-app-password"
mail:
  to: home-recipient@example.com
github:
  token: home-token
`)

		cfg, err := Load("")
		if err != nil {
			t.Fatalf(`Load("") returned error: %v`, err)
		}
		if cfg.GmailAccount != "home-user@example.com" {
			t.Errorf(`Load("") did not fall back to ~/.config: got GmailAccount=%q`, cfg.GmailAccount)
		}
	})
}

// TestLoad_FileNotFoundErrorNamesPath covers the Design step 4 requirement:
// an unreadable config file produces an error naming the file path (the
// path itself is not a secret).
func TestLoad_FileNotFoundErrorNamesPath(t *testing.T) {
	dir := t.TempDir()
	// no config.yaml written

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load() returned nil error for a missing config file")
	}
	wantPath := filepath.Join(dir, "config.yaml")
	if !strings.Contains(err.Error(), wantPath) {
		t.Errorf("Load() error = %q, want it to contain path %q", err.Error(), wantPath)
	}
}

// TestLoad_MalformedYAMLErrorNamesPath covers the Design step 4 requirement
// for malformed YAML.
func TestLoad_MalformedYAMLErrorNamesPath(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "gmail: [this is not valid: yaml")

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load() returned nil error for malformed YAML")
	}
	wantPath := filepath.Join(dir, "config.yaml")
	if !strings.Contains(err.Error(), wantPath) {
		t.Errorf("Load() error = %q, want it to contain path %q", err.Error(), wantPath)
	}
}
