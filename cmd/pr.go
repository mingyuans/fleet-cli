package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/executor"
	"github.com/xq-yan/fleet-cli/internal/git"
	"github.com/xq-yan/fleet-cli/internal/manifest"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/workspace"
)

// prAlreadyExistsErr indicates a PR already exists for the branch.
type prAlreadyExistsErr struct {
	URL string
}

func (e *prAlreadyExistsErr) Error() string {
	return "PR already exists: " + e.URL
}

// noCommitsErr indicates there are no commits between the base and head branches.
type noCommitsErr struct{}

func (e *noCommitsErr) Error() string {
	return "no commits between branches"
}

var prTitle string

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Push and create pull request",
	RunE:  runPR,
}

func init() {
	prCmd.Flags().StringVarP(&prTitle, "title", "t", "", "pull request title (default: branch name)")
	rootCmd.AddCommand(prCmd)
}

func runPR(cmd *cobra.Command, args []string) error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found, install it from https://cli.github.com/")
	}

	output.Header("Creating PRs for %d projects...", len(projects))

	results := executor.Run(projects, ws.SyncJ, func(proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
		return prProject(ws.Root, proj, log)
	})

	counts := executor.CountResults(results)
	output.Summary(counts, []string{"created", "exists", "skipped", "failed"})

	return nil
}

func prProject(root string, proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
	projDir, branch, remote, label, status, message, ok := pushPreflight(root, proj, false)
	if !ok {
		return label, status, message
	}

	// Push first
	log("pushing %s -> %s ...", branch, remote)
	if err := git.Push(projDir, remote, branch); err != nil {
		return "failed", executor.StatusFail, "push failed: " + err.Error()
	}

	// Resolve upstream repo (fetch remote) for PR target
	fetchURL := proj.CloneURL
	_, upstreamRepo, ok := git.ParseRepoOwner(fetchURL)
	if !ok {
		return "failed", executor.StatusFail, "cannot parse upstream repo from: " + fetchURL
	}

	// Resolve base branch
	baseBranch := proj.Revision
	fetchRemote := resolveRemote(projDir, proj.Remote)
	if fetchRemote != "" {
		if resolved := resolveRevision(projDir, fetchRemote, proj.Revision, proj.MasterMainCompat); resolved != "" {
			baseBranch = resolved
		}
	}

	// Determine --head: for fork, need "fork-owner:branch"
	head := branch
	if proj.IsForkPush() {
		_, pushRepo, pushOK := git.ParseRepoOwner(proj.PushURL)
		if pushOK {
			pushOwner := strings.SplitN(pushRepo, "/", 2)[0]
			head = pushOwner + ":" + branch
		}
	}

	title := prTitle
	if title == "" {
		title = branch
	}

	// Create PR via gh
	log("creating PR: %s -> %s/%s ...", head, upstreamRepo, baseBranch)
	prURL, err := ghCreatePR(projDir, upstreamRepo, baseBranch, head, title)
	if err != nil {
		var existsErr *prAlreadyExistsErr
		if errors.As(err, &existsErr) {
			return "exists " + existsErr.URL, executor.StatusSuccess, ""
		}
		var ncErr *noCommitsErr
		if errors.As(err, &ncErr) {
			return "skipped", executor.StatusSkip, "no commits"
		}
		return "failed", executor.StatusFail, err.Error()
	}

	return "created " + prURL, executor.StatusSuccess, ""
}

func ghCreatePR(dir, repo, base, head, title string) (string, error) {
	cmd := exec.Command("gh", "pr", "create",
		"--repo", repo,
		"--base", base,
		"--head", head,
		"--title", title,
		"--body", "",
	)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			return "", fmt.Errorf("gh pr create: %w", err)
		}
		if strings.Contains(errMsg, "already exists") {
			if url := extractPRURL(errMsg); url != "" {
				return "", &prAlreadyExistsErr{URL: url}
			}
		}
		if strings.Contains(errMsg, "No commits between") {
			return "", &noCommitsErr{}
		}
		return "", fmt.Errorf("gh pr create: %s", errMsg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// extractPRURL extracts the PR URL from a gh CLI "already exists" error message.
func extractPRURL(errMsg string) string {
	for _, line := range strings.Split(errMsg, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "https://") {
			return line
		}
	}
	return ""
}
