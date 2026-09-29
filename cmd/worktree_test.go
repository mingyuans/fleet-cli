package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupWorktreeWorkspace builds a workspace with a root project (path ".") and a
// nested one (path "services/api"), both cloned from local bare repos. localXML
// is written as local_fleet.xml when non-empty, which lets a test exercise the
// merge path as well as the resolution path.
func setupWorktreeWorkspace(t *testing.T, defaultAttrs, localXML string) string {
	t.Helper()
	dir := t.TempDir()

	remotesDir := filepath.Join(dir, "remotes", "Org")
	if err := os.MkdirAll(remotesDir, 0755); err != nil {
		t.Fatal(err)
	}
	createBareRepo(t, remotesDir, "root")
	createBareRepo(t, remotesDir, "api")

	wsDir := filepath.Join(dir, "workspace")
	runGit(t, "git", "clone", filepath.Join(remotesDir, "root.git"), wsDir, "--origin", "github")
	runGit(t, "git", "clone", filepath.Join(remotesDir, "api.git"),
		filepath.Join(wsDir, "services", "api"), "--origin", "github")

	manifest := `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github" fetch="` + remotesDir + `/" />
  <default remote="github" revision="master" sync-j="2" ` + defaultAttrs + ` />
  <project name="root" path="." />
  <project name="api" path="services/api" />
</manifest>`
	if err := os.WriteFile(filepath.Join(wsDir, "fleet.xml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	localPath := ""
	if localXML != "" {
		localPath = filepath.Join(wsDir, "local_fleet.xml")
		if err := os.WriteFile(localPath, []byte(localXML), 0644); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("FLEET_MANIFEST", filepath.Join(wsDir, "fleet.xml"))
	t.Setenv("FLEET_LOCAL_MANIFEST", localPath)
	groupFilter = ""
	worktreeBranch = ""
	worktreeRevision = ""
	worktreeDest = ""

	return wsDir
}

func runGit(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
}

func runWorktreeCmd(t *testing.T, args ...string) {
	t.Helper()
	rootCmd.SetArgs(append([]string{"worktree"}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("worktree %v: %v", args, err)
	}
}

// chdir switches the process working directory for the duration of the test so a
// test can prove the resolved paths do not depend on it.
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

func assertIsWorktree(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		t.Fatalf("expected a worktree at %s: %v", path, err)
	}
}

// TestWorktreeRelativeBaseTargetsWorkspaceRoot is the core regression test: with a
// relative worktree-base, the nested project's worktree must land under the
// workspace-root-anchored worktree root, not inside the project's own repository.
func TestWorktreeRelativeBaseTargetsWorkspaceRoot(t *testing.T) {
	wsDir := setupWorktreeWorkspace(t, `worktree-base="./worktrees/fleet"`, "")
	chdir(t, wsDir)

	runWorktreeCmd(t, "feat-x")

	assertIsWorktree(t, filepath.Join(wsDir, "worktrees", "fleet", "feat-x", "services", "api"))

	// The bug this guards against created the worktree at
	// services/api/worktrees/fleet/feat-x/services/api.
	stray := filepath.Join(wsDir, "services", "api", "worktrees")
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Errorf("expected no worktree nested inside the project repo, found %s", stray)
	}
}

func TestWorktreeRootProjectPath(t *testing.T) {
	wsDir := setupWorktreeWorkspace(t, `worktree-base="./worktrees/fleet"`, "")
	chdir(t, wsDir)

	runWorktreeCmd(t, "feat-x")

	assertIsWorktree(t, filepath.Join(wsDir, "worktrees", "fleet", "feat-x"))
}

// TestWorktreeIsIdempotent covers the second half of the old failure: because the
// existence check and git disagreed on where the worktree was, a re-run reported
// a failure instead of a skip.
func TestWorktreeIsIdempotent(t *testing.T) {
	wsDir := setupWorktreeWorkspace(t, `worktree-base="./worktrees/fleet"`, "")
	chdir(t, wsDir)

	runWorktreeCmd(t, "feat-x")
	runWorktreeCmd(t, "feat-x")

	assertIsWorktree(t, filepath.Join(wsDir, "worktrees", "fleet", "feat-x", "services", "api"))
}

func TestWorktreeIndependentOfWorkingDirectory(t *testing.T) {
	wsDir := setupWorktreeWorkspace(t, `worktree-base="./worktrees/fleet"`, "")
	chdir(t, filepath.Join(wsDir, "services", "api"))

	runWorktreeCmd(t, "feat-y")

	assertIsWorktree(t, filepath.Join(wsDir, "worktrees", "fleet", "feat-y"))
	assertIsWorktree(t, filepath.Join(wsDir, "worktrees", "fleet", "feat-y", "services", "api"))

	stray := filepath.Join(wsDir, "services", "api", "worktrees")
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Errorf("expected no worktree nested inside the project repo, found %s", stray)
	}
}

// TestWorktreeCopyFromLocalManifest exercises the merge fix and the copy target
// together: worktree-base and worktree-copy are declared only in local_fleet.xml.
func TestWorktreeCopyFromLocalManifest(t *testing.T) {
	localXML := `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <default worktree-base="./worktrees/fleet" worktree-copy=".env" />
</manifest>`
	wsDir := setupWorktreeWorkspace(t, "", localXML)
	chdir(t, wsDir)

	apiDir := filepath.Join(wsDir, "services", "api")
	if err := os.WriteFile(filepath.Join(apiDir, ".env"), []byte("TOKEN=x\n"), 0644); err != nil {
		t.Fatal(err)
	}

	runWorktreeCmd(t, "feat-x")

	copied := filepath.Join(wsDir, "worktrees", "fleet", "feat-x", "services", "api", ".env")
	if _, err := os.Stat(copied); err != nil {
		t.Errorf("expected worktree-copy to place the file at %s: %v", copied, err)
	}
}

func TestWorktreeRelativeDest(t *testing.T) {
	wsDir := setupWorktreeWorkspace(t, "", "")
	chdir(t, filepath.Join(wsDir, "services", "api"))

	runWorktreeCmd(t, "--dest", "./tmp/wt", "-b", "feat-dest")

	assertIsWorktree(t, filepath.Join(wsDir, "tmp", "wt"))
	assertIsWorktree(t, filepath.Join(wsDir, "tmp", "wt", "services", "api"))
}
