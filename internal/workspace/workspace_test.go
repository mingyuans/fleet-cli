package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestWorkspace(t *testing.T, defaultXML, localXML string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fleet.xml"), []byte(defaultXML), 0644); err != nil {
		t.Fatal(err)
	}
	if localXML != "" {
		if err := os.WriteFile(filepath.Join(dir, "local_fleet.xml"), []byte(localXML), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const testDefaultXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github" fetch="git@github.com:Org/" />
  <default remote="github" revision="master" sync-j="4" />
  <project name="svc-a" path="services/svc-a" groups="core" />
  <project name="svc-b" path="services/svc-b" />
</manifest>`

const testLocalXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="fork" fetch="git@github.com:user/" />
  <default push="fork" />
</manifest>`

func TestLoadWithEnvManifest(t *testing.T) {
	dir := setupTestWorkspace(t, testDefaultXML, "")
	t.Setenv("FLEET_MANIFEST", filepath.Join(dir, "fleet.xml"))
	t.Setenv("FLEET_LOCAL_MANIFEST", "")

	ws, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.Root != dir {
		t.Errorf("expected root=%s, got %s", dir, ws.Root)
	}
	if len(ws.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(ws.Projects))
	}
	if ws.HasLocalManifest {
		t.Error("expected HasLocalManifest=false")
	}
}

func TestLoadWithLocalManifest(t *testing.T) {
	dir := setupTestWorkspace(t, testDefaultXML, testLocalXML)
	t.Setenv("FLEET_MANIFEST", filepath.Join(dir, "fleet.xml"))
	t.Setenv("FLEET_LOCAL_MANIFEST", "")

	ws, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ws.HasLocalManifest {
		t.Error("expected HasLocalManifest=true")
	}
	if len(ws.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(ws.Projects))
	}
	if !ws.Projects[0].HasPushRemote {
		t.Error("expected project to have push remote after merge")
	}
}

func TestLoadWithLocalManifestEnv(t *testing.T) {
	dir := setupTestWorkspace(t, testDefaultXML, "")
	localDir := t.TempDir()
	localPath := filepath.Join(localDir, "custom_local.xml")
	if err := os.WriteFile(localPath, []byte(testLocalXML), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FLEET_MANIFEST", filepath.Join(dir, "fleet.xml"))
	t.Setenv("FLEET_LOCAL_MANIFEST", localPath)

	ws, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ws.HasLocalManifest {
		t.Error("expected HasLocalManifest=true with FLEET_LOCAL_MANIFEST")
	}
}

func TestLoadFromParentDir(t *testing.T) {
	dir := setupTestWorkspace(t, testDefaultXML, "")
	subDir := filepath.Join(dir, "services", "svc-a")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Clear env vars and change to subdir
	t.Setenv("FLEET_MANIFEST", "")
	t.Setenv("FLEET_LOCAL_MANIFEST", "")
	oldDir, _ := os.Getwd()
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })

	ws, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Resolve symlinks for macOS where /var -> /private/var
	resolvedDir, _ := filepath.EvalSymlinks(dir)
	resolvedRoot, _ := filepath.EvalSymlinks(ws.Root)
	if resolvedRoot != resolvedDir {
		t.Errorf("expected root=%s, got %s", resolvedDir, resolvedRoot)
	}
}

func TestLoadMissingManifest(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLEET_MANIFEST", "")
	t.Setenv("FLEET_LOCAL_MANIFEST", "")
	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
}

func TestResolvePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available")
	}

	tests := []struct {
		name string
		base string
		path string
		want string
	}{
		{"empty", "/w", "", ""},
		{"relative", "/w", "worktrees/fleet", "/w/worktrees/fleet"},
		{"dot-relative", "/w", "./worktrees/fleet", "/w/worktrees/fleet"},
		{"parent-relative", "/w/sub", "../worktrees", "/w/worktrees"},
		{"absolute", "/w", "/abs/worktrees", "/abs/worktrees"},
		{"tilde", "/w", "~/worktrees/fleet", filepath.Join(home, "worktrees/fleet")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolvePath(tt.base, tt.path); got != tt.want {
				t.Errorf("ResolvePath(%q, %q) = %q, want %q", tt.base, tt.path, got, tt.want)
			}
		})
	}
}

const testWorktreeBaseXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github" fetch="git@github.com:Org/" />
  <default remote="github" revision="master" />
  <project name="svc-a" path="services/svc-a" />
</manifest>`

const testWorktreeBaseLocalXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <default worktree-base="./worktrees/fleet" worktree-copy=".env,.env.*" />
</manifest>`

// TestLoadResolvesRelativeWorktreeBase covers the two halves of the fix together:
// worktree-base declared only in local_fleet.xml must survive the merge, and the
// relative path must resolve against the workspace root rather than the process
// working directory.
func TestLoadResolvesRelativeWorktreeBase(t *testing.T) {
	dir := setupTestWorkspace(t, testWorktreeBaseXML, testWorktreeBaseLocalXML)
	subDir := filepath.Join(dir, "services", "svc-a")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FLEET_MANIFEST", "")
	t.Setenv("FLEET_LOCAL_MANIFEST", "")
	oldDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldDir) })

	for _, cwd := range []string{dir, subDir} {
		if err := os.Chdir(cwd); err != nil {
			t.Fatal(err)
		}
		ws, err := Load()
		if err != nil {
			t.Fatalf("unexpected error from %s: %v", cwd, err)
		}
		if !filepath.IsAbs(ws.WorktreeBase) {
			t.Fatalf("expected absolute worktree base from %s, got %q", cwd, ws.WorktreeBase)
		}
		// Compare against ws.Root, which carries the same symlink form as the
		// resolved base, so no EvalSymlinks normalization is needed.
		want := filepath.Join(ws.Root, "worktrees", "fleet")
		if ws.WorktreeBase != want {
			t.Errorf("worktree base from cwd %s = %q, want %q", cwd, ws.WorktreeBase, want)
		}
		if len(ws.Projects) != 1 || len(ws.Projects[0].WorktreeCopy) != 2 {
			t.Errorf("expected local worktree-copy patterns to merge, got %v", ws.Projects[0].WorktreeCopy)
		}
	}
}

func TestLoadEmptyWorktreeBaseStaysEmpty(t *testing.T) {
	dir := setupTestWorkspace(t, testWorktreeBaseXML, "")
	t.Setenv("FLEET_MANIFEST", filepath.Join(dir, "fleet.xml"))
	t.Setenv("FLEET_LOCAL_MANIFEST", "")

	ws, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.WorktreeBase != "" {
		t.Errorf("expected empty worktree base to stay empty, got %q", ws.WorktreeBase)
	}
}
