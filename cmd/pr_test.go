package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseBranchCandidates(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single branch",
			input: "testing",
			want:  []string{"testing"},
		},
		{
			name:  "two branches",
			input: "testing-incy|testing",
			want:  []string{"testing-incy", "testing"},
		},
		{
			name:  "three branches",
			input: "staging|testing|main",
			want:  []string{"staging", "testing", "main"},
		},
		{
			name:  "with whitespace",
			input: " testing | main ",
			want:  []string{"testing", "main"},
		},
		{
			name:  "empty segments filtered",
			input: "|testing|",
			want:  []string{"testing"},
		},
		{
			name:  "all empty segments",
			input: "||",
			want:  nil,
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace only segments",
			input: " | | ",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBranchCandidates(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseBranchCandidates(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseBranchCandidates(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestResolveBaseFromCandidates(t *testing.T) {
	// Set up a bare repo with specific branches
	dir := t.TempDir()
	bareRepo := filepath.Join(dir, "upstream.git")

	cmds := [][]string{
		{"git", "init", "--bare", bareRepo},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("create bare repo: %v\n%s", err, out)
		}
	}

	// Clone, create branches, push
	cloneDir := filepath.Join(dir, "clone")
	setupCmds := [][]string{
		{"git", "clone", bareRepo, cloneDir},
		{"git", "-C", cloneDir, "config", "user.email", "test@test.com"},
		{"git", "-C", cloneDir, "config", "user.name", "Test"},
		{"git", "-C", cloneDir, "commit", "--allow-empty", "-m", "initial"},
		{"git", "-C", cloneDir, "push", "origin", "master"},
		{"git", "-C", cloneDir, "checkout", "-b", "testing"},
		{"git", "-C", cloneDir, "push", "origin", "testing"},
	}
	for _, c := range setupCmds {
		cmd := exec.Command(c[0], c[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup: %v\n%s", err, out)
		}
	}

	// Create a working repo that fetches from the bare repo
	workDir := filepath.Join(dir, "work")
	workCmds := [][]string{
		{"git", "clone", bareRepo, workDir},
		{"git", "-C", workDir, "fetch", "origin"},
	}
	for _, c := range workCmds {
		cmd := exec.Command(c[0], c[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("work setup: %v\n%s", err, out)
		}
	}

	tests := []struct {
		name        string
		candidates  []string
		wantBranch  string
	}{
		{
			name:       "single candidate exists",
			candidates: []string{"testing"},
			wantBranch: "testing",
		},
		{
			name:       "first candidate exists",
			candidates: []string{"master", "testing"},
			wantBranch: "master",
		},
		{
			name:       "first not exist, second exists",
			candidates: []string{"nonexist", "testing"},
			wantBranch: "testing",
		},
		{
			name:       "all not exist",
			candidates: []string{"nonexist1", "nonexist2"},
			wantBranch: "",
		},
		{
			name:       "empty candidates",
			candidates: []string{},
			wantBranch: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveBaseFromCandidates(workDir, "origin", tt.candidates)
			if got != tt.wantBranch {
				t.Errorf("resolveBaseFromCandidates(%v) = %q, want %q", tt.candidates, got, tt.wantBranch)
			}
		})
	}

	// Clean up
	os.RemoveAll(cloneDir)
}
