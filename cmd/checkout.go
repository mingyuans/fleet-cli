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

var checkoutFrom string

var checkoutCmd = &cobra.Command{
	Use:   "checkout <branch>",
	Short: "Switch to a branch across all repos, optionally from a collaborator's fork",
	Long: "Switch to a branch across all repos.\n\n" +
		"Without --from, behaves like `fleet start`: creates/switches the branch from\n" +
		"the upstream default branch.\n\n" +
		"With --from <user>, adds (or reuses) a git remote pointing at <user>'s fork,\n" +
		"fetches it, and switches to <user>/<branch> — handy for debugging a collaborator's\n" +
		"branch locally. The fork remote is kept so later `fleet sync` can track it.",
	Args: cobra.ExactArgs(1),
	RunE: runCheckout,
}

func init() {
	checkoutCmd.Flags().StringVar(&checkoutFrom, "from", "", "collaborator GitHub username whose fork to check out from")
	rootCmd.AddCommand(checkoutCmd)
}

func runCheckout(cmd *cobra.Command, args []string) error {
	branch := args[0]

	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}
	if checkoutFrom != "" {
		output.Info("From fork: %s", output.Bold(checkoutFrom))
	}
	output.Header("Checking out branch %s across %d projects...", output.Bold(branch), len(projects))

	results := executor.Run(projects, ws.SyncJ, func(proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
		if checkoutFrom == "" {
			return startProject(ws.Root, proj, branch, log)
		}
		return checkoutForkProject(ws.Root, proj, branch, checkoutFrom, log)
	})

	counts := executor.CountResults(results)
	output.Summary(counts, []string{"created", "checked-out", "switched", "skipped", "failed"})

	return nil
}

func checkoutForkProject(root string, proj manifest.ResolvedProject, branch, user string, log executor.LogFunc) (string, executor.ResultStatus, string) {
	projDir := filepath.Join(root, proj.Path)

	if _, err := os.Stat(projDir); os.IsNotExist(err) {
		return "skipped", executor.StatusSkip, "not cloned"
	}

	// Derive the collaborator fork URL from the fetch remote URL.
	forkURL, ok := git.DeriveForkURL(proj.CloneURL, user)
	if !ok {
		return "failed", executor.StatusFail, "cannot derive fork URL from " + proj.CloneURL
	}

	// Ensure the collaborator remote exists and points at the right URL.
	if !git.RemoteExists(projDir, user) {
		log("adding fork remote %s ...", user)
		if err := git.RemoteAdd(projDir, user, forkURL); err != nil {
			return "failed", executor.StatusFail, err.Error()
		}
	} else if current, err := git.RemoteGetURL(projDir, user); err != nil {
		return "failed", executor.StatusFail, err.Error()
	} else if current != forkURL {
		log("fixing fork remote %s URL ...", user)
		if err := git.RemoteSetURL(projDir, user, forkURL); err != nil {
			return "failed", executor.StatusFail, err.Error()
		}
	}

	// Fetch from the collaborator remote.
	log("fetching %s ...", user)
	if err := git.Fetch(projDir, user); err != nil {
		return "failed", executor.StatusFail, "fetch failed: " + err.Error()
	}

	targetRef := user + "/" + branch
	if !git.RemoteRefExists(projDir, targetRef) {
		return "skipped", executor.StatusSkip, targetRef + " not found"
	}

	// Decide the local branch name based on the upstream source, not just the name.
	// If a same-named local branch already tracks a different source, use a
	// fork-qualified local branch name to avoid clobbering it.
	localBranch := branch
	if git.BranchExists(projDir, branch) {
		if upstream, _ := git.BranchUpstream(projDir, branch); upstream != targetRef {
			localBranch = targetRef
		}
	}

	// Already on the target branch and tracking the target source.
	current, err := git.CurrentBranch(projDir)
	if err != nil {
		return "failed", executor.StatusFail, err.Error()
	}
	if current == localBranch {
		if upstream, _ := git.BranchUpstream(projDir, localBranch); upstream == targetRef {
			return "skipped", executor.StatusSkip, "already tracking " + targetRef
		}
	}

	// Switch to the existing local branch, or create it from the fork ref.
	if git.BranchExists(projDir, localBranch) {
		log("switching to existing branch %s ...", localBranch)
		if err := git.CheckoutBranch(projDir, localBranch); err != nil {
			return "failed", executor.StatusFail, err.Error()
		}
		return "switched", executor.StatusSuccess, ""
	}

	startPoint := "refs/remotes/" + targetRef
	log("creating branch %s from %s ...", localBranch, targetRef)
	if err := git.CreateBranchFrom(projDir, localBranch, startPoint); err != nil {
		return "failed", executor.StatusFail, err.Error()
	}
	return "checked-out from " + targetRef, executor.StatusSuccess, ""
}
