package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) { // AC-1
	dir := t.TempDir()
	s := NewStore(dir)

	if err := s.Save("v1.2.3"); err != nil {
		t.Fatalf("Save: unexpected error: %v", err)
	}

	version, found, err := s.Load()
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("Load: found = false, want true")
	}
	if version != "v1.2.3" {
		t.Fatalf("Load: version = %q, want %q", version, "v1.2.3")
	}
}

func TestLoadNoStateFile(t *testing.T) { // AC-2
	dir := t.TempDir()
	s := NewStore(dir)

	version, found, err := s.Load()
	if err != nil {
		t.Fatalf("Load: unexpected error for absent state file: %v", err)
	}
	if found {
		t.Fatalf("Load: found = true, want false")
	}
	if version != "" {
		t.Fatalf("Load: version = %q, want empty string", version)
	}
}

func TestLoadMalformedStateFile(t *testing.T) { // AC-3
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("setup: writing malformed state file: %v", err)
	}
	s := NewStore(dir)

	_, found, err := s.Load()
	if err == nil {
		t.Fatalf("Load: expected error for malformed state file, got nil")
	}
	if found {
		t.Fatalf("Load: found = true on error path, want false")
	}
}

func TestSaveCreatesStateDirectory(t *testing.T) { // AC-4
	dir := filepath.Join(t.TempDir(), "nested", "state")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("setup: expected %s not to exist yet", dir)
	}
	s := NewStore(dir)

	if err := s.Save("v1.0.0"); err != nil {
		t.Fatalf("Save: unexpected error: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("Save: state directory was not created: err=%v", err)
	}
}

func TestSaveLeavesNoTempResidue(t *testing.T) { // AC-5
	dir := t.TempDir()
	s := NewStore(dir)

	if err := s.Save("v1.0.0"); err != nil {
		t.Fatalf("Save (1): unexpected error: %v", err)
	}
	if err := s.Save("v2.0.0"); err != nil {
		t.Fatalf("Save (2): unexpected error: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("directory listing = %v, want exactly one entry (no temp-file residue)", names)
	}
	if entries[0].Name() != "state.json" {
		t.Fatalf("directory entry = %q, want %q", entries[0].Name(), "state.json")
	}

	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var payload struct {
		LastVersion string `json:"last_version"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if payload.LastVersion != "v2.0.0" {
		t.Fatalf("last_version = %q, want %q (final content only, no partial write)", payload.LastVersion, "v2.0.0")
	}
}

func TestDefaultStateDirXDGResolution(t *testing.T) { // AC-6
	t.Run("XDG_STATE_HOME set", func(t *testing.T) {
		xdgHome := t.TempDir()
		t.Setenv("XDG_STATE_HOME", xdgHome)

		s := NewStore("")
		if err := s.Save("v1.0.0"); err != nil {
			t.Fatalf("Save: unexpected error: %v", err)
		}

		want := filepath.Join(xdgHome, "claude-release-notes", "state.json")
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("expected state file at %s: %v", want, err)
		}
	})

	t.Run("XDG_STATE_HOME unset falls back to home-relative default", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("XDG_STATE_HOME", "")
		t.Setenv("HOME", home)

		s := NewStore("")
		if err := s.Save("v1.0.0"); err != nil {
			t.Fatalf("Save: unexpected error: %v", err)
		}

		want := filepath.Join(home, ".local", "state", "claude-release-notes", "state.json")
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("expected state file at %s: %v", want, err)
		}
	})
}
