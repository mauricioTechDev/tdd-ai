package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/macosta/tdd-ai/internal/types"
)

const DefaultFileName = ".tdd-ai.json"

// FilePath returns the session file path for a given directory.
func FilePath(dir string) string {
	return filepath.Join(dir, DefaultFileName)
}

// Exists checks whether a session file exists in the given directory.
func Exists(dir string) bool {
	_, err := os.Stat(FilePath(dir))
	return err == nil
}

// Create initializes a new session and saves it to disk.
func Create(dir string) (*types.Session, error) {
	return CreateWithMode(dir, types.ModeGreenfield)
}

// CreateWithMode initializes a new session with the specified mode and saves it.
func CreateWithMode(dir string, mode types.Mode) (*types.Session, error) {
	s := types.NewSession()
	s.Mode = mode
	if err := Save(dir, s); err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}
	return s, nil
}

// Load reads a session from the given directory.
func Load(dir string) (*types.Session, error) {
	data, err := os.ReadFile(FilePath(dir))
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}

	var s types.Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}
	return &s, nil
}

// Save writes the session state to disk.
func Save(dir string, s *types.Session) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding session: %w", err)
	}
	if err := os.WriteFile(FilePath(dir), data, 0644); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}
	return nil
}

// LoadOrFail loads a session and returns a user-friendly error if none exists.
func LoadOrFail(dir string) (*types.Session, error) {
	if !Exists(dir) {
		return nil, fmt.Errorf("no TDD session found. Run 'tdd-ai init' first")
	}
	return Load(dir)
}
