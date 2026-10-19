// Package claims turns a plain-language claim about a command into checks that
// can run on every agent loop.
//
// The loop has two speeds. Verifying a claim is slow and done once: tdd-ai runs
// the command under a tracer and records what it touched, and the agent judges
// whether that evidence, and the files the claim depends on, back the claim.
// Checking is fast and done constantly: tdd-ai distills the accepted evidence
// into deterministic checks (pinned file hashes, a read/write/network policy,
// the exit code and the expected output) that rerun without any model in CI
// and in every agent loop.
package claims

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	// Dir holds the claims file and expected outputs. Commit it with the code.
	Dir = ".tdd-ai"
	// FileName is the claims file inside Dir.
	FileName = "claims.json"
)

// Status is where a claim is in the verify-once, check-always loop.
type Status string

const (
	StatusUnverified Status = "unverified" // added, never observed
	StatusObserved   Status = "observed"   // observed, waiting for a verdict
	StatusVerified   Status = "verified"   // judged and distilled into checks
)

// Claim is a plain-language statement about what a command does.
type Claim struct {
	ID       int       `json:"id"`
	Text     string    `json:"text"`
	Run      []string  `json:"run"`
	Pins     []string  `json:"pins,omitempty"`
	Status   Status    `json:"status"`
	AddedAt  string    `json:"added_at"`
	Evidence *Evidence `json:"evidence,omitempty"`
	Checks   *Checks   `json:"checks,omitempty"`
	// Note is the verifier's judgment of why the evidence backs the claim.
	Note string `json:"note,omitempty"`
}

// Evidence is what tdd-ai observed one run of a claim's command do.
// Project paths are relative to the project root; paths outside it start
// with "~/" (home) or "/". Runtime and system files are left out.
type Evidence struct {
	ObservedAt     string            `json:"observed_at"`
	Observer       string            `json:"observer"`
	ExitCode       int               `json:"exit_code"`
	StdoutSHA256   string            `json:"stdout_sha256"`
	StdoutBytes    int               `json:"stdout_bytes"`
	Reads          []string          `json:"reads"`
	Writes         []string          `json:"writes"`
	ExternalReads  []string          `json:"external_reads"`
	ExternalWrites []string          `json:"external_writes"`
	Listed         []string          `json:"listed,omitempty"`
	Network        []string          `json:"network"`
	Execs          []string          `json:"execs"`
	Pins           map[string]string `json:"pins,omitempty"`
}

// Checks are the deterministic assertions distilled from accepted evidence.
type Checks struct {
	DistilledAt string `json:"distilled_at"`
	// Run is the command that was verified; a check fails if the claim's command changes.
	Run      []string          `json:"run"`
	Pins     map[string]string `json:"pins,omitempty"`
	Reads    []string          `json:"reads"`
	Writes   []string          `json:"writes"`
	External []string          `json:"external"`
	Network  []string          `json:"network"`
	Execs    []string          `json:"execs"`
	ExitCode int               `json:"exit_code"`
	// Stdout is the project-relative file holding the expected output.
	Stdout       string `json:"stdout,omitempty"`
	StdoutSHA256 string `json:"stdout_sha256,omitempty"`
}

// File is the claims file.
type File struct {
	Claims []Claim `json:"claims"`
	NextID int     `json:"next_id"`
}

// Path returns the claims file path for a project directory.
func Path(dir string) string {
	return filepath.Join(dir, Dir, FileName)
}

// ObservedOutputPath is where verify keeps a claim's latest output, relative to the project.
func ObservedOutputPath(id int) string {
	return path.Join(Dir, "observed", fmt.Sprintf("%d.out", id))
}

// ExpectedOutputPath is where distill keeps a claim's expected output, relative to the project.
func ExpectedOutputPath(id int) string {
	return path.Join(Dir, "expected", fmt.Sprintf("%d.out", id))
}

// Load reads the claims file, or returns an empty one if there is none yet.
func Load(dir string) (*File, error) {
	data, err := os.ReadFile(Path(dir))
	if errors.Is(err, os.ErrNotExist) {
		return &File{Claims: []Claim{}, NextID: 1}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading claims file: %w", err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing claims file: %w", err)
	}
	if f.Claims == nil {
		f.Claims = []Claim{}
	}
	if f.NextID < 1 {
		f.NextID = 1
	}
	return &f, nil
}

// Save writes the claims file.
func Save(dir string, f *File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding claims: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, Dir), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", Dir, err)
	}
	if err := os.WriteFile(Path(dir), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing claims file: %w", err)
	}
	return nil
}

// Add appends a new, unverified claim and returns it.
func (f *File) Add(text string, run, pins []string, now time.Time) *Claim {
	f.Claims = append(f.Claims, Claim{
		ID:      f.NextID,
		Text:    text,
		Run:     run,
		Pins:    pins,
		Status:  StatusUnverified,
		AddedAt: now.UTC().Format(time.RFC3339),
	})
	f.NextID++
	return &f.Claims[len(f.Claims)-1]
}

// Get returns the claim with the given ID.
func (f *File) Get(id int) (*Claim, error) {
	for i := range f.Claims {
		if f.Claims[i].ID == id {
			return &f.Claims[i], nil
		}
	}
	return nil, fmt.Errorf("claim %d not found", id)
}

// CleanPin normalizes a pinned path and rejects paths outside the project.
func CleanPin(p string) (string, error) {
	if p == "" || filepath.IsAbs(p) {
		return "", fmt.Errorf("pin %q must be a path inside the project", p)
	}
	clean := filepath.ToSlash(filepath.Clean(p))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("pin %q must be a path inside the project", p)
	}
	return clean, nil
}

// HashFile returns the sha256 of a project-relative file.
func HashFile(dir, rel string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		return "", err
	}
	return HashBytes(data), nil
}

// HashBytes returns the hex sha256 of b.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// HashPins hashes every pinned file. Missing files hash to "" so callers can
// decide whether that is an error (verify) or a failed check (check).
func HashPins(dir string, pins []string) map[string]string {
	out := make(map[string]string, len(pins))
	for _, p := range pins {
		h, err := HashFile(dir, p)
		if err != nil {
			h = ""
		}
		out[p] = h
	}
	return out
}
