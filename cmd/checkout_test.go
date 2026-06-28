package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xq-yan/fleet-cli/internal/git"
)

// TestCheckoutWithoutFromDelegatesToStart verifies that `fleet checkout <branch>`
// without --from behaves like `fleet start`: it creates the branch from the
// upstream default branch.
func TestCheckoutWithoutFromDelegatesToStart(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	t.Setenv("FLEET_MANIFEST", filepath.Join(wsDir, "fleet.xml"))
	t.Setenv("FLEET_LOCAL_MANIFEST", "")
	groupFilter = ""
	checkoutFrom = ""

	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	rootCmd.SetArgs([]string{"checkout", "feature/x"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("checkout failed: %v", err)
	}

	svcDir := filepath.Join(wsDir, "services", "svc-a")
	branch, err := git.CurrentBranch(svcDir)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "feature/x" {
		t.Errorf("expected branch feature/x, got %q", branch)
	}
}

// TestCheckoutFromForkSkipsNotCloned verifies that repos which are not cloned
// are skipped and the command completes without error.
func TestCheckoutFromForkSkipsNotCloned(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	t.Setenv("FLEET_MANIFEST", filepath.Join(wsDir, "fleet.xml"))
	t.Setenv("FLEET_LOCAL_MANIFEST", "")
	groupFilter = ""
	checkoutFrom = ""

	// Do NOT init: repo stays uncloned.
	svcDir := filepath.Join(wsDir, "services", "svc-a")
	if _, err := os.Stat(svcDir); !os.IsNotExist(err) {
		t.Fatalf("expected svc-a to be absent before test")
	}

	rootCmd.SetArgs([]string{"checkout", "feature/x", "--from", "alice"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("checkout should not error on skipped repos: %v", err)
	}

	// Reset the package-level flag to avoid leaking into other tests.
	checkoutFrom = ""
}
