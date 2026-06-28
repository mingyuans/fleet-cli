package cmd

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xq-yan/fleet-cli/internal/executor"
	"github.com/xq-yan/fleet-cli/internal/git"
	"github.com/xq-yan/fleet-cli/internal/manifest"
)

// makeAliasRepo creates a bare upstream with a "master" branch plus the given
// extra branches, then a work clone whose "origin" points at it. It returns the
// work directory. The work clone has NOT fetched the extra branches yet; callers
// relying on remote refs should fetch first (startProject does this itself).
func makeAliasRepo(t *testing.T, extraBranches ...string) string {
	t.Helper()
	dir := t.TempDir()
	bareRepo := filepath.Join(dir, "upstream.git")
	seedDir := filepath.Join(dir, "seed")

	cmds := [][]string{
		{"git", "init", "--bare", bareRepo},
		{"git", "clone", bareRepo, seedDir},
		{"git", "-C", seedDir, "config", "user.email", "test@test.com"},
		{"git", "-C", seedDir, "config", "user.name", "Test"},
		{"git", "-C", seedDir, "commit", "--allow-empty", "-m", "initial"},
		{"git", "-C", seedDir, "branch", "-m", "master"},
		{"git", "-C", seedDir, "push", "origin", "master"},
	}
	for _, b := range extraBranches {
		cmds = append(cmds,
			[]string{"git", "-C", seedDir, "checkout", "-b", b, "master"},
			[]string{"git", "-C", seedDir, "commit", "--allow-empty", "-m", "on " + b},
			[]string{"git", "-C", seedDir, "push", "origin", b},
		)
	}

	workDir := filepath.Join(dir, "work")
	cmds = append(cmds, []string{"git", "clone", bareRepo, workDir})

	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %v\n%s", c, err, out)
		}
	}
	return workDir
}

func TestBranchAliasGroup(t *testing.T) {
	groups := [][]string{{"testing", "testing-incy"}, {"master", "main"}}

	if g := branchAliasGroup("testing-incy", groups); len(g) != 2 || g[0] != "testing" {
		t.Errorf("expected testing group, got %v", g)
	}
	if g := branchAliasGroup("main", groups); len(g) != 2 || g[0] != "master" {
		t.Errorf("expected master group, got %v", g)
	}
	if g := branchAliasGroup("feature-x", groups); g != nil {
		t.Errorf("expected nil for non-member, got %v", g)
	}
}

func TestResolveBranchWithAliases(t *testing.T) {
	// Upstream has testing (but not testing-incy) and master.
	workDir := makeAliasRepo(t, "testing")
	if err := git.Fetch(workDir, "origin"); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	groups := [][]string{{"testing-incy", "testing"}, {"master", "main"}}

	tests := []struct {
		name   string
		branch string
		want   string
	}{
		{"branch itself exists", "testing", "testing"},
		{"fallback to alias member", "testing-incy", "testing"},
		{"master exists directly", "master", "master"},
		{"fallback main->master", "main", "master"},
		{"no member exists", "staging", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveBranchWithAliases(workDir, "origin", tt.branch, groups)
			if got != tt.want {
				t.Errorf("resolveBranchWithAliases(%q) = %q, want %q", tt.branch, got, tt.want)
			}
		})
	}
}

// branchStartedFrom reports whether local branch tip equals origin/<expected> tip.
func branchStartedFrom(t *testing.T, dir, branch, expectedRemoteBranch string) bool {
	t.Helper()
	local, err := exec.Command("git", "-C", dir, "rev-parse", branch).Output()
	if err != nil {
		t.Fatalf("rev-parse %s: %v", branch, err)
	}
	remote, err := exec.Command("git", "-C", dir, "rev-parse", "origin/"+expectedRemoteBranch).Output()
	if err != nil {
		t.Fatalf("rev-parse origin/%s: %v", expectedRemoteBranch, err)
	}
	return string(local) == string(remote)
}

// TestPRBaseDoesNotStackAlias verifies that --base resolution (resolveBaseFromCandidates)
// never auto-falls-back to an alias-group sibling: only the explicit candidates are tried.
func TestPRBaseDoesNotStackAlias(t *testing.T) {
	// Upstream has testing but NOT testing-incy; both would be in the same alias group.
	workDir := makeAliasRepo(t, "testing")
	if err := git.Fetch(workDir, "origin"); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	// --base testing-incy: testing-incy is absent; resolveBaseFromCandidates must NOT
	// fall back to the sibling "testing", so it returns "".
	if got := resolveBaseFromCandidates(workDir, "origin", []string{"testing-incy"}); got != "" {
		t.Errorf("resolveBaseFromCandidates([testing-incy]) = %q, want \"\" (no alias fallback)", got)
	}
	// Explicit pipe fallback still works, but that is the user's own choice.
	if got := resolveBaseFromCandidates(workDir, "origin", []string{"testing-incy", "testing"}); got != "testing" {
		t.Errorf("resolveBaseFromCandidates([testing-incy testing]) = %q, want testing", got)
	}
}

func TestStartProjectAliasBehavior(t *testing.T) {
	noop := executor.LogFunc(func(string, ...any) {})
	groups := [][]string{{"testing-incy", "testing"}}

	t.Run("target alias exists on remote", func(t *testing.T) {
		work := makeAliasRepo(t, "testing", "testing-incy")
		proj := manifest.ResolvedProject{Path: ".", Remote: "origin", Revision: "master", AliasGroups: groups}
		label, status, msg := startProject(work, proj, "testing-incy", noop)
		if status != executor.StatusSuccess {
			t.Fatalf("status=%v msg=%q label=%q", status, msg, label)
		}
		if !branchStartedFrom(t, work, "testing-incy", "testing-incy") {
			t.Error("expected testing-incy based on origin/testing-incy")
		}
	})

	t.Run("target alias missing falls back", func(t *testing.T) {
		work := makeAliasRepo(t, "testing") // no testing-incy on remote
		proj := manifest.ResolvedProject{Path: ".", Remote: "origin", Revision: "master", AliasGroups: groups}
		label, status, msg := startProject(work, proj, "testing-incy", noop)
		if status != executor.StatusSuccess {
			t.Fatalf("status=%v msg=%q label=%q", status, msg, label)
		}
		// Local branch name must be the user input, based on origin/testing.
		if cur, _ := git.CurrentBranch(work); cur != "testing-incy" {
			t.Errorf("current branch = %q, want testing-incy", cur)
		}
		if !branchStartedFrom(t, work, "testing-incy", "testing") {
			t.Error("expected testing-incy based on origin/testing")
		}
	})

	t.Run("no alias member fails", func(t *testing.T) {
		work := makeAliasRepo(t) // only master
		proj := manifest.ResolvedProject{Path: ".", Remote: "origin", Revision: "master", AliasGroups: groups}
		_, status, _ := startProject(work, proj, "testing-incy", noop)
		if status != executor.StatusFail {
			t.Fatalf("expected fail, got %v", status)
		}
	})

	t.Run("non-alias branch uses revision", func(t *testing.T) {
		work := makeAliasRepo(t, "testing")
		proj := manifest.ResolvedProject{Path: ".", Remote: "origin", Revision: "master", AliasGroups: groups}
		_, status, msg := startProject(work, proj, "my-feature", noop)
		if status != executor.StatusSuccess {
			t.Fatalf("status=%v msg=%q", status, msg)
		}
		if !branchStartedFrom(t, work, "my-feature", "master") {
			t.Error("expected my-feature based on origin/master")
		}
	})

	t.Run("local branch already exists", func(t *testing.T) {
		work := makeAliasRepo(t, "testing")
		// Pre-create a local testing-incy branch.
		if out, err := exec.Command("git", "-C", work, "branch", "testing-incy").CombinedOutput(); err != nil {
			t.Fatalf("pre-create branch: %v\n%s", err, out)
		}
		proj := manifest.ResolvedProject{Path: ".", Remote: "origin", Revision: "master", AliasGroups: groups}
		label, status, msg := startProject(work, proj, "testing-incy", noop)
		if status != executor.StatusSuccess {
			t.Fatalf("status=%v msg=%q label=%q", status, msg, label)
		}
		if cur, _ := git.CurrentBranch(work); cur != "testing-incy" {
			t.Errorf("current branch = %q, want testing-incy", cur)
		}
	})
}
