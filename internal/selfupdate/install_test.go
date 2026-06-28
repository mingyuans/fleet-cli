package selfupdate

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceBinary(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "fleet")

	if err := os.WriteFile(dest, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("seed dest: %v", err)
	}

	newContent := []byte("new binary content")
	if err := ReplaceBinary(dest, newContent); err != nil {
		t.Fatalf("ReplaceBinary error: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if !bytes.Equal(got, newContent) {
		t.Fatalf("dest = %q, want %q", got, newContent)
	}

	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat dest: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("dest is not executable, mode = %v", info.Mode())
	}

	// No leftover temp files should remain in the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 file in dir, found %d", len(entries))
	}
}

func TestReplaceBinaryNoWritePermission(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission checks do not apply")
	}

	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	if err := os.Mkdir(roDir, 0o555); err != nil {
		t.Fatalf("mkdir readonly: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(roDir, 0o755) })

	dest := filepath.Join(roDir, "fleet")
	err := ReplaceBinary(dest, []byte("new binary"))
	if err == nil {
		t.Fatal("expected permission error, got nil")
	}
}
