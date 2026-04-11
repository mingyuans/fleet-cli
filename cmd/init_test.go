package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xq-yan/fleet-cli/internal/git"
)

// createBareRepo creates a bare git repo to serve as a remote.
func createBareRepo(t *testing.T, dir, name string) string {
	t.Helper()
	repoDir := filepath.Join(dir, name+".git")
	cmds := [][]string{
		{"git", "init", "--bare", repoDir},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("create bare repo %s: %v\n%s", name, err, out)
		}
	}

	// Create a temporary clone to make an initial commit
	tmpClone := filepath.Join(dir, "tmp-"+name)
	for _, c := range [][]string{
		{"git", "clone", repoDir, tmpClone},
		{"git", "-C", tmpClone, "config", "user.email", "test@test.com"},
		{"git", "-C", tmpClone, "config", "user.name", "Test"},
		{"git", "-C", tmpClone, "commit", "--allow-empty", "-m", "initial"},
		{"git", "-C", tmpClone, "push", "origin", "master"},
	} {
		cmd := exec.Command(c[0], c[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("init commit for %s: %v\n%s", name, err, out)
		}
	}
	os.RemoveAll(tmpClone)

	return repoDir
}

// setupIntegrationWorkspace creates a test workspace with bare repos and manifest files.
func setupIntegrationWorkspace(t *testing.T) (workspaceDir string, cleanup func()) {
	t.Helper()
	dir := t.TempDir()

	// Create bare repos as "remotes"
	remotesDir := filepath.Join(dir, "remotes", "Org")
	forkDir := filepath.Join(dir, "remotes", "User")
	os.MkdirAll(remotesDir, 0755)
	os.MkdirAll(forkDir, 0755)

	createBareRepo(t, remotesDir, "svc-a")
	createBareRepo(t, forkDir, "svc-a")

	workspaceDir = filepath.Join(dir, "workspace")
	os.MkdirAll(workspaceDir, 0755)

	defaultXML := `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github" fetch="` + remotesDir + `/svc-a" />
  <remote name="fork" fetch="` + forkDir + `/svc-a" />
  <default remote="github" revision="master" sync-j="2" push="fork" />
  <project name="" path="services/svc-a" />
</manifest>`

	// Note: we use empty name since our fetch URLs already point to the bare repo
	// Let's use a simpler approach with the full URL as fetch
	defaultXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github" fetch="` + remotesDir + `/" />
  <remote name="fork" fetch="` + forkDir + `/" />
  <default remote="github" revision="master" sync-j="2" push="fork" />
  <project name="svc-a" path="services/svc-a" />
</manifest>`

	if err := os.WriteFile(filepath.Join(workspaceDir, "fleet.xml"), []byte(defaultXML), 0644); err != nil {
		t.Fatal(err)
	}

	return workspaceDir, func() {}
}

func TestInitCloneAndIdempotent(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	t.Setenv("FLEET_MANIFEST", filepath.Join(wsDir, "fleet.xml"))
	t.Setenv("FLEET_LOCAL_MANIFEST", "")

	// Reset group filter
	groupFilter = ""

	// First init: should clone
	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	svcDir := filepath.Join(wsDir, "services", "svc-a")
	if _, err := os.Stat(svcDir); err != nil {
		t.Fatalf("expected svc-a to be cloned: %v", err)
	}

	// Verify github remote exists
	if !git.RemoteExists(svcDir, "github") {
		t.Error("expected github remote to exist")
	}

	// Verify fork remote exists
	if !git.RemoteExists(svcDir, "fork") {
		t.Error("expected fork remote to exist")
	}

	// Second init: should skip (idempotent)
	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second init failed: %v", err)
	}

	// Verify still works
	if !git.RemoteExists(svcDir, "github") {
		t.Error("expected github remote still exists after second init")
	}
}
