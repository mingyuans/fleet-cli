package cmd

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/executor"
	"github.com/xq-yan/fleet-cli/internal/git"
	"github.com/xq-yan/fleet-cli/internal/manifest"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/workspace"
)

var finishRemote bool

var finishCmd = &cobra.Command{
	Use:   "finish <branch>",
	Short: "Delete a branch and switch back to the default branch",
	Args:  cobra.ExactArgs(1),
	RunE:  runFinish,
}

func init() {
	finishCmd.Flags().BoolVarP(&finishRemote, "remote", "r", false, "also delete the branch on push remote")
	rootCmd.AddCommand(finishCmd)
}

func runFinish(cmd *cobra.Command, args []string) error {
	branch := args[0]

	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}
	output.Header("Finishing branch %s across %d projects...", output.Bold(branch), len(projects))

	results := executor.Run(projects, ws.SyncJ, func(proj manifest.ResolvedProject, buf *bytes.Buffer, log executor.LogFunc) (string, executor.ResultStatus, string) {
		return finishProject(ws.Root, proj, branch, log)
	})

	counts := executor.CountResults(results)
	output.Summary(counts, []string{"finished", "skipped", "failed"})

	return nil
}

func finishProject(root string, proj manifest.ResolvedProject, branch string, log executor.LogFunc) (string, executor.ResultStatus, string) {
	projDir := filepath.Join(root, proj.Path)

	if _, err := os.Stat(projDir); os.IsNotExist(err) {
		return "skipped", executor.StatusSkip, "not cloned"
	}

	if !git.BranchExists(projDir, branch) {
		return "skipped", executor.StatusSkip, "branch " + branch + " not found"
	}

	// If currently on the target branch, switch to default branch first
	current, err := git.CurrentBranch(projDir)
	if err != nil {
		return "failed", executor.StatusFail, err.Error()
	}

	if current == branch {
		defaultBranch := proj.Revision
		// Use compat to find the actual default branch
		remote := resolveRemote(projDir, proj.Remote)
		if remote != "" {
			if resolved := resolveRevision(projDir, remote, proj.Revision, proj.MasterMainCompat); resolved != "" {
				defaultBranch = resolved
			}
		}

		log("switching to %s ...", defaultBranch)
		if err := git.CheckoutBranch(projDir, defaultBranch); err != nil {
			return "failed", executor.StatusFail, "cannot switch to " + defaultBranch + ": " + err.Error()
		}
	}

	log("deleting local branch %s ...", branch)
	if err := git.DeleteBranch(projDir, branch); err != nil {
		return "failed", executor.StatusFail, err.Error()
	}

	if finishRemote && proj.HasPushRemote && git.RemoteExists(projDir, proj.Push) {
		log("deleting remote branch %s/%s ...", proj.Push, branch)
		if err := git.DeleteRemoteBranch(projDir, proj.Push, branch); err != nil {
			return "failed", executor.StatusFail, "local deleted, remote failed: " + err.Error()
		}
	}

	return "finished", executor.StatusSuccess, ""
}
