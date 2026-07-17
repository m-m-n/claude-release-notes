// Package state persists the last delivered release version so that
// subsequent runs know where to resume from.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	stateFileName            = "state.json"
	appDirName               = "claude-release-notes"
	defaultStateHomeRelative = ".local/state"
	tempFilePattern          = ".state-*.tmp"
)

// Store persists the last delivered release version under a directory.
type Store struct {
	dir string
}

// stateFile is the on-disk JSON shape (SPEC.md: `{"last_version": "v1.2.3"}`).
type stateFile struct {
	LastVersion string `json:"last_version"`
}

// NewStore creates a Store rooted at stateDir. When stateDir is empty, the
// XDG state directory is resolved internally: $XDG_STATE_HOME (when set) or
// ~/.local/state (fallback), plus "claude-release-notes/".
func NewStore(stateDir string) *Store {
	if stateDir == "" {
		stateDir = defaultStateDir()
	}
	return &Store{dir: stateDir}
}

func defaultStateDir() string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, defaultStateHomeRelative)
	}
	return filepath.Join(base, appDirName)
}

func (s *Store) path() string {
	return filepath.Join(s.dir, stateFileName)
}

// Load reads the stored version. When no state file exists yet, it returns
// (found=false, err=nil) rather than an error. A present-but-corrupt file is
// reported as an error, distinct from absence.
func (s *Store) Load() (version string, found bool, err error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("state: reading %s: %w", s.path(), err)
	}

	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return "", false, fmt.Errorf("state: parsing %s: %w", s.path(), err)
	}

	return sf.LastVersion, true, nil
}

// Save persists version atomically: it writes to a temp file in the same
// directory and renames it into place, so a concurrent reader always sees
// either the old or the new content, never a partial write. Missing
// directories are created as needed.
func (s *Store) Save(version string) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("state: creating directory %s: %w", s.dir, err)
	}

	data, err := json.Marshal(stateFile{LastVersion: version})
	if err != nil {
		return fmt.Errorf("state: encoding state: %w", err)
	}

	tmp, err := os.CreateTemp(s.dir, tempFilePattern)
	if err != nil {
		return fmt.Errorf("state: creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("state: writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("state: closing temp file: %w", err)
	}

	if err := os.Rename(tmpName, s.path()); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("state: renaming temp file into place: %w", err)
	}

	return nil
}
