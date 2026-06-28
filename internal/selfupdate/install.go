package selfupdate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// CurrentExecutable resolves the absolute path of the running fleet binary,
// following symlinks so we replace the real file rather than a link to it.
func CurrentExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate current executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		// Fall back to the unresolved path if the symlink cannot be evaluated.
		return exe, nil
	}
	return resolved, nil
}

// ReplaceBinary atomically replaces the executable at destPath with newBinary.
// It writes to a temporary file in the same directory (so os.Rename stays on
// one filesystem and is atomic), sets the executable bit, then renames over the
// target. On a permission failure it returns an error hinting at sudo.
func ReplaceBinary(destPath string, newBinary []byte) error {
	dir := filepath.Dir(destPath)

	tmp, err := os.CreateTemp(dir, ".fleet-update-*")
	if err != nil {
		if isPermission(err) {
			return permissionError(dir)
		}
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup if we fail before the final rename.
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(newBinary); err != nil {
		tmp.Close()
		return fmt.Errorf("write new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close new binary: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("set executable permission: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		if isPermission(err) {
			return permissionError(dir)
		}
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

// isPermission reports whether err is (or wraps) a permission-denied error.
func isPermission(err error) bool {
	return errors.Is(err, fs.ErrPermission)
}

// permissionError returns a user-facing error suggesting elevated privileges.
func permissionError(dir string) error {
	return fmt.Errorf("no write permission for %s; re-run with elevated privileges (e.g. sudo fleet update)", dir)
}
