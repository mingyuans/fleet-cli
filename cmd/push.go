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

var pushAll bool

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push current branch to push remote",
	RunE:  runPush,
}

func init() {
	pushCmd.Flags().BoolVar(&pushAll, "all", false, "push all branches including default")
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)

	// Show push remote if configured
	for _, p := range projects {
		if p.HasPushRemote {
			output.Info("Push remote: %s", output.Bold(p.Push))
			break
		}
	}
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}
	output.Header("Pushing %d projects...", len(projects))

	results := executor.Run(projects, ws.SyncJ, func(proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
		return pushProject(ws.Root, proj, log)
	})

	counts := executor.CountResults(results)
	output.Summary(counts, []string{"pushed", "skipped", "failed"})

	return nil
}

// pushPreflight validates that a project is ready for push/pr operations.
// Returns the project directory, current branch, and push remote name.
// If validation fails, returns a skip/fail result tuple.
func pushPreflight(root string, proj manifest.ResolvedProject, allowDefault bool) (projDir, branch, remote string, label string, status executor.ResultStatus, message string, ok bool) {
	projDir = filepath.Join(root, proj.Path)

	if _, err := os.Stat(projDir); os.IsNotExist(err) {
		return "", "", "", "skipped", executor.StatusSkip, "not cloned", false
	}

	branch, err := git.CurrentBranch(projDir)
	if err != nil {
		return "", "", "", "failed", executor.StatusFail, err.Error(), false
	}
	if branch == "" {
		return "", "", "", "skipped", executor.StatusSkip, "detached HEAD", false
	}
	if !allowDefault && branch == proj.Revision {
		return "", "", "", "skipped", executor.StatusSkip, "on default branch", false
	}

	if !proj.HasPushRemote {
		return "", "", "", "skipped", executor.StatusSkip, "no push remote configured", false
	}
	if !git.RemoteExists(projDir, proj.Push) {
		return "", "", "", "skipped", executor.StatusSkip, "push remote " + proj.Push + " not found, run 'fleet init'", false
	}

	return projDir, branch, proj.Push, "", "", "", true
}

func pushProject(root string, proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
	projDir, branch, remote, label, status, message, ok := pushPreflight(root, proj, pushAll)
	if !ok {
		return label, status, message
	}

	log("pushing %s -> %s ...", branch, remote)
	if err := git.Push(projDir, remote, branch); err != nil {
		return "failed", executor.StatusFail, err.Error()
	}

	return "pushed", executor.StatusSuccess, ""
}
