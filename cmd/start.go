package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/executor"
	"github.com/xq-yan/fleet-cli/internal/git"
	"github.com/xq-yan/fleet-cli/internal/manifest"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/workspace"
)

var startCmd = &cobra.Command{
	Use:   "start <branch>",
	Short: "Create and switch to a new branch based on upstream default branch",
	Args:  cobra.ExactArgs(1),
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	branch := args[0]

	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}
	output.Header("Starting branch %s across %d projects...", output.Bold(branch), len(projects))

	results := executor.Run(projects, ws.SyncJ, func(proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
		return startProject(ws.Root, proj, branch, log)
	})

	counts := executor.CountResults(results)
	output.Summary(counts, []string{"created", "switched", "skipped", "failed"})

	return nil
}

func startProject(root string, proj manifest.ResolvedProject, branch string, log executor.LogFunc) (string, executor.ResultStatus, string) {
	projDir := filepath.Join(root, proj.Path)

	if _, err := os.Stat(projDir); os.IsNotExist(err) {
		return "skipped", executor.StatusSkip, "not cloned"
	}

	// If already on the target branch, skip
	current, err := git.CurrentBranch(projDir)
	if err != nil {
		return "failed", executor.StatusFail, err.Error()
	}
	if current == branch {
		return "skipped", executor.StatusSkip, "already on " + branch
	}

	// If the local branch already exists, just switch to it
	if git.BranchExists(projDir, branch) {
		log("switching to existing branch %s ...", branch)
		if err := git.CheckoutBranch(projDir, branch); err != nil {
			return "failed", executor.StatusFail, err.Error()
		}
		return "switched", executor.StatusSuccess, ""
	}

	// Resolve the remote to use
	remote := resolveRemote(projDir, proj.Remote)
	if remote == "" {
		return "failed", executor.StatusFail, "no suitable remote found (tried " + proj.Remote + " and origin)"
	}

	// Fetch latest from remote
	log("fetching %s ...", remote)
	if err := git.Fetch(projDir, remote); err != nil {
		return "failed", executor.StatusFail, "fetch failed: " + err.Error()
	}

	// Resolve the base branch to create from.
	// When the target branch itself belongs to an alias group (e.g. testing-incy),
	// base it on that group (testing-incy -> testing fallback) rather than the
	// configured revision. Otherwise use the configured revision (alias fallback
	// still applies, e.g. master<->main).
	var baseBranch string
	if group := branchAliasGroup(branch, proj.AliasGroups); group != nil {
		baseBranch = resolveBranchWithAliases(projDir, remote, branch, proj.AliasGroups)
		if baseBranch == "" {
			return "failed", executor.StatusFail, "no alias branch found on " + remote + " (tried " + strings.Join(group, ", ") + ")"
		}
	} else {
		baseBranch = resolveRevision(projDir, remote, proj.Revision, proj.AliasGroups)
		if baseBranch == "" {
			return "failed", executor.StatusFail, remote + "/" + proj.Revision + " not found"
		}
	}

	startPoint := remote + "/" + baseBranch
	log("creating branch %s from %s ...", branch, startPoint)
	if err := git.CreateBranchFrom(projDir, branch, startPoint); err != nil {
		return "failed", executor.StatusFail, err.Error()
	}

	return "created from " + remote + "/" + baseBranch, executor.StatusSuccess, ""
}
