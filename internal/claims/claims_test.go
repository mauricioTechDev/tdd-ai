package claims

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestLoadWithoutAFileReturnsAnEmptyOne(t *testing.T) {
	f, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(f.Claims) != 0 || f.NextID != 1 {
		t.Errorf("Load = %+v, want no claims and NextID 1", f)
	}
}

func TestAddSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	f, _ := Load(dir)
	first := f.Add("first claim", []string{"node", "a.mjs"}, []string{"a.mjs"}, now)
	second := f.Add("second claim", []string{"true"}, nil, now)
	if first.ID != 1 || second.ID != 2 || first.Status != StatusUnverified {
		t.Fatalf("ids/status = %d, %d, %s", first.ID, second.ID, first.Status)
	}
	if err := Save(dir, f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".tdd-ai", "claims.json")); err != nil {
		t.Fatalf("claims file not written: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(loaded, f) {
		t.Errorf("round trip =\n%+v\nwant\n%+v", loaded, f)
	}
	c, err := loaded.Get(2)
	if err != nil || c.Text != "second claim" || c.AddedAt != "2026-10-19T12:00:00Z" {
		t.Errorf("Get(2) = %+v, %v", c, err)
	}
	if _, err := loaded.Get(9); err == nil {
		t.Error("Get(9) should fail")
	}
}

func TestLoadRejectsACorruptFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, Dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(dir), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Error("Load should fail on a corrupt claims file")
	}
}

func TestCleanPin(t *testing.T) {
	good := map[string]string{
		"skills/invoice/SKILL.md":   "skills/invoice/SKILL.md",
		"./skills//invoice/x.mjs":   "skills/invoice/x.mjs",
		"skills/../skills/SKILL.md": "skills/SKILL.md",
	}
	for in, want := range good {
		if got, err := CleanPin(in); err != nil || got != want {
			t.Errorf("CleanPin(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", ".", "..", "../outside.txt", "/etc/passwd"} {
		if _, err := CleanPin(bad); err == nil {
			t.Errorf("CleanPin(%q) should fail", bad)
		}
	}
}

func TestHashPinsMarksMissingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := HashPins(dir, []string{"a.txt", "missing.txt"})
	want := map[string]string{
		"a.txt":       "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		"missing.txt": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("HashPins = %v, want %v", got, want)
	}
}

func TestOutputPaths(t *testing.T) {
	if got := ObservedOutputPath(3); got != ".tdd-ai/observed/3.out" {
		t.Errorf("ObservedOutputPath = %q", got)
	}
	if got := ExpectedOutputPath(3); got != ".tdd-ai/expected/3.out" {
		t.Errorf("ExpectedOutputPath = %q", got)
	}
}
