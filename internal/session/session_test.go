package session

import (
	"os"
	"testing"

	"github.com/macosta/tdd-ai/internal/types"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tdd-ai-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestCreateAndLoad(t *testing.T) {
	dir := tempDir(t)

	created, err := Create(dir)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if created.Phase != types.PhaseRed {
		t.Errorf("created session phase = %q, want %q", created.Phase, types.PhaseRed)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Phase != types.PhaseRed {
		t.Errorf("loaded session phase = %q, want %q", loaded.Phase, types.PhaseRed)
	}
	if loaded.NextID != 1 {
		t.Errorf("loaded session NextID = %d, want 1", loaded.NextID)
	}
}

func TestExists(t *testing.T) {
	dir := tempDir(t)

	if Exists(dir) {
		t.Error("Exists() should be false before Create()")
	}

	_, _ = Create(dir)

	if !Exists(dir) {
		t.Error("Exists() should be true after Create()")
	}
}

func TestSavePersistsChanges(t *testing.T) {
	dir := tempDir(t)

	s, _ := Create(dir)
	s.AddSpec("test feature")
	s.Phase = types.PhaseGreen

	if err := Save(dir, s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Phase != types.PhaseGreen {
		t.Errorf("loaded phase = %q, want %q", loaded.Phase, types.PhaseGreen)
	}
	if len(loaded.Specs) != 1 {
		t.Fatalf("loaded specs length = %d, want 1", len(loaded.Specs))
	}
	if loaded.Specs[0].Description != "test feature" {
		t.Errorf("loaded spec description = %q, want %q", loaded.Specs[0].Description, "test feature")
	}
}

func TestLoadOrFailNoSession(t *testing.T) {
	dir := tempDir(t)

	_, err := LoadOrFail(dir)
	if err == nil {
		t.Error("LoadOrFail() should return error when no session exists")
	}
}

func TestLoadOrFailWithSession(t *testing.T) {
	dir := tempDir(t)
	_, _ = Create(dir)

	s, err := LoadOrFail(dir)
	if err != nil {
		t.Fatalf("LoadOrFail() error: %v", err)
	}
	if s.Phase != types.PhaseRed {
		t.Errorf("session phase = %q, want %q", s.Phase, types.PhaseRed)
	}
}

func TestSavePersistsTestCmd(t *testing.T) {
	dir := tempDir(t)

	s, _ := Create(dir)
	s.TestCmd = "go test ./..."
	s.LastTestResult = "pass"

	if err := Save(dir, s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.TestCmd != "go test ./..." {
		t.Errorf("loaded TestCmd = %q, want %q", loaded.TestCmd, "go test ./...")
	}
	if loaded.LastTestResult != "pass" {
		t.Errorf("loaded LastTestResult = %q, want %q", loaded.LastTestResult, "pass")
	}
}

func TestSavePersistsHistory(t *testing.T) {
	dir := tempDir(t)

	s, _ := Create(dir)
	s.AddEvent("test_run", func(e *types.Event) {
		e.Result = "pass"
	})
	s.AddEvent("phase_next", func(e *types.Event) {
		e.From = "red"
		e.To = "green"
	})

	if err := Save(dir, s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded.History) != 2 {
		t.Fatalf("loaded History length = %d, want 2", len(loaded.History))
	}
	if loaded.History[0].Action != "test_run" {
		t.Errorf("History[0].Action = %q, want %q", loaded.History[0].Action, "test_run")
	}
	if loaded.History[0].Result != "pass" {
		t.Errorf("History[0].Result = %q, want %q", loaded.History[0].Result, "pass")
	}
	if loaded.History[1].From != "red" {
		t.Errorf("History[1].From = %q, want %q", loaded.History[1].From, "red")
	}
	if loaded.History[1].To != "green" {
		t.Errorf("History[1].To = %q, want %q", loaded.History[1].To, "green")
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	dir := tempDir(t)
	os.WriteFile(FilePath(dir), []byte("not json"), 0644)

	_, err := Load(dir)
	if err == nil {
		t.Error("Load() should return error for corrupted file")
	}
}
