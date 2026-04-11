package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/executor"
	"github.com/xq-yan/fleet-cli/internal/git"
	"github.com/xq-yan/fleet-cli/internal/manifest"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/workspace"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync repositories from upstream",
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}
	output.Header("Syncing %d projects...", len(projects))

	results := executor.Run(projects, ws.SyncJ, func(proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
		return syncProject(ws.Root, proj, log)
	})

	counts := executor.CountResults(results)
	output.Summary(counts, []string{"rebased", "fetched", "skipped", "failed"})

	return nil
}

func syncProject(root string, proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
	projDir := filepath.Join(root, proj.Path)

	if _, err := os.Stat(projDir); os.IsNotExist(err) {
		return "skipped", executor.StatusSkip, "not cloned"
	}

	remote := resolveRemote(projDir, proj.Remote)
	if remote == "" {
		return "failed", executor.StatusFail, "no suitable remote found (tried " + proj.Remote + " and origin)"
	}

	branch, err := git.CurrentBranch(projDir)
	if err != nil {
		return "failed", executor.StatusFail, err.Error()
	}

	// Check if on default branch (with master<->main compat)
	revision := resolveRevision(projDir, remote, proj.Revision, proj.MasterMainCompat)
	if revision == "" {
		revision = proj.Revision
	}

	if branch == revision {
		log("pulling --rebase %s/%s ...", remote, revision)
		if err := git.PullRebase(projDir, remote, revision); err != nil {
			return "failed", executor.StatusFail, "rebase conflict or error: " + err.Error()
		}
		return "rebased", executor.StatusSuccess, ""
	}

	log("fetching %s (on branch %s) ...", remote, branch)
	if err := git.Fetch(projDir, remote); err != nil {
		return "failed", executor.StatusFail, err.Error()
	}
	return "fetched", executor.StatusSuccess, ""
}
