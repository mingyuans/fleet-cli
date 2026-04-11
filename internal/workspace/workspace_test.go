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
  <project name="svc-a" path="services/svc-a" groups="feed" />
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
