package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %s\n%s", c, err, out)
		}
	}
	return dir
}

func TestCurrentBranch(t *testing.T) {
	dir := initTestRepo(t)
	branch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default branch could be main or master depending on git config
	if branch == "" {
		t.Error("expected non-empty branch name")
	}
}

func TestCurrentBranchDetached(t *testing.T) {
	dir := initTestRepo(t)
	cmd := exec.Command("git", "checkout", "--detach")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("detach: %s\n%s", err, out)
	}

	branch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "" {
		t.Errorf("expected empty branch for detached HEAD, got %q", branch)
	}
}

func TestStatusPorcelain(t *testing.T) {
	dir := initTestRepo(t)

	out, err := StatusPorcelain(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected clean status, got %q", out)
	}

	// Create untracked file
	if err := os.WriteFile(filepath.Join(dir, "newfile.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err = StatusPorcelain(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == "" {
		t.Error("expected dirty status")
	}
}

func TestRemoteOperations(t *testing.T) {
	dir := initTestRepo(t)

	if RemoteExists(dir, "upstream") {
		t.Error("expected upstream to not exist")
	}

	if err := RemoteAdd(dir, "upstream", "https://example.com/repo.git"); err != nil {
		t.Fatalf("RemoteAdd: %v", err)
	}

	if !RemoteExists(dir, "upstream") {
		t.Error("expected upstream to exist after add")
	}

	url, err := RemoteGetURL(dir, "upstream")
	if err != nil {
		t.Fatalf("RemoteGetURL: %v", err)
	}
	if url != "https://example.com/repo.git" {
		t.Errorf("expected URL https://example.com/repo.git, got %q", url)
	}

	if err := RemoteSetURL(dir, "upstream", "https://example.com/new.git"); err != nil {
		t.Fatalf("RemoteSetURL: %v", err)
	}
	url, err = RemoteGetURL(dir, "upstream")
	if err != nil {
		t.Fatalf("RemoteGetURL: %v", err)
	}
	if url != "https://example.com/new.git" {
		t.Errorf("expected new URL, got %q", url)
	}
}

func TestConfigSet(t *testing.T) {
	dir := initTestRepo(t)
	if err := ConfigSet(dir, "user.name", "FleetTest"); err != nil {
		t.Fatalf("ConfigSet: %v", err)
	}
	out, err := output(dir, "git", "config", "user.name")
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if out != "FleetTest" {
		t.Errorf("expected FleetTest, got %q", out)
	}
}
