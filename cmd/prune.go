package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/executor"
	"github.com/xq-yan/fleet-cli/internal/git"
	"github.com/xq-yan/fleet-cli/internal/manifest"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/workspace"
)

// protectedBranches are never deleted by prune under any circumstances.
var protectedBranches = []string{"master", "main", "develop", "testing"}

// pruneCandidate holds a branch eligible for deletion in a specific project.
type pruneCandidate struct {
	proj   manifest.ResolvedProject
	branch string
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Find and delete local branches already merged into the revision branch",
	Long: `prune finds local feature branches that have been fully merged into the
upstream revision branch. It lists the candidates and asks for confirmation
before deleting them locally and on the remote.

The following branches are never deleted: master, main, develop, testing.`,
	RunE: runPrune,
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}

func runPrune(cmd *cobra.Command, args []string) error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}
	output.Header("Scanning %d project(s) for merged branches...", len(projects))

	candidates := collectPruneCandidates(ws.Root, projects, ws.SyncJ)

	if len(candidates) == 0 {
		fmt.Println()
		output.Info("No merged branches found.")
		return nil
	}

	fmt.Println()
	output.Info("The following merged branches can be safely deleted:")
	fmt.Println()

	table := output.NewTable("PROJECT", "BRANCH")
	for _, c := range candidates {
		table.AddRow([]string{c.proj.Path, c.branch}, []string{"", output.ColorYellow})
	}
	table.Print()

	fmt.Println()
	fmt.Printf("Delete %d branch(es) (local + remote)? [y/N] ", len(candidates))

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if answer := strings.TrimSpace(scanner.Text()); !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
		fmt.Println()
		output.Info("Aborted.")
		return nil
	}

	fmt.Println()
	output.Header("Deleting %d branch(es)...", len(candidates))

	deleted, failed := deleteCandidates(ws.Root, candidates)

	fmt.Println()
	if deleted > 0 {
		output.Success("%d branch(es) deleted.", deleted)
	}
	if failed > 0 {
		output.Warning("%d branch(es) failed to delete.", failed)
	}

	return nil
}

// collectPruneCandidates fetches each project in parallel and returns branches merged into its revision.
func collectPruneCandidates(root string, projects []manifest.ResolvedProject, concurrency int) []pruneCandidate {
	var mu sync.Mutex
	var candidates []pruneCandidate

	executor.Run(projects, concurrency, func(proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
		projDir := filepath.Join(root, proj.Path)

		if _, err := os.Stat(projDir); os.IsNotExist(err) {
			return "skipped", executor.StatusSkip, "not cloned"
		}

		remote := resolveRemote(projDir, proj.Remote)
		if remote == "" {
			return "failed", executor.StatusFail, "no suitable remote found (tried " + proj.Remote + " and origin)"
		}

		log("fetching %s ...", remote)
		if err := git.Fetch(projDir, remote); err != nil {
			return "failed", executor.StatusFail, "fetch failed: " + err.Error()
		}

		revision := resolveRevision(projDir, remote, proj.Revision, proj.MasterMainCompat)
		if revision == "" {
			return "failed", executor.StatusFail, "revision " + proj.Revision + " not found on remote"
		}

		branches, err := git.ListMergedBranches(projDir, remote+"/"+revision)
		if err != nil {
			return "failed", executor.StatusFail, "cannot list merged branches: " + err.Error()
		}

		// Skip the revision branch itself in addition to the global protected list.
		skip := append([]string{revision}, protectedBranches...)
		var found []pruneCandidate
		for _, branch := range branches {
			if !slices.Contains(skip, branch) {
				found = append(found, pruneCandidate{proj: proj, branch: branch})
			}
		}

		if len(found) == 0 {
			return "clean", executor.StatusSkip, ""
		}

		mu.Lock()
		candidates = append(candidates, found...)
		mu.Unlock()

		return fmt.Sprintf("%d merged", len(found)), executor.StatusSuccess, ""
	})

	return candidates
}

// deleteCandidates deletes each candidate branch locally and on the remote.
// Returns counts of deleted and failed branches.
func deleteCandidates(root string, candidates []pruneCandidate) (deleted, failed int) {
	for _, c := range candidates {
		projDir := filepath.Join(root, c.proj.Path)

		remote := resolveRemote(projDir, c.proj.Remote)

		current, err := git.CurrentBranch(projDir)
		if err != nil {
			output.Error("%s/%s: cannot determine current branch: %s", c.proj.Path, c.branch, err)
			failed++
			continue
		}
		if current == c.branch {
			revision := resolveRevision(projDir, remote, c.proj.Revision, c.proj.MasterMainCompat)
			if revision == "" {
				revision = c.proj.Revision
			}
			if err := git.CheckoutBranch(projDir, revision); err != nil {
				output.Error("%s/%s: cannot switch away to %s: %s", c.proj.Path, c.branch, revision, err)
				failed++
				continue
			}
		}

		if err := git.DeleteBranch(projDir, c.branch); err != nil {
			output.Error("%s/%s: local delete failed: %s", c.proj.Path, c.branch, err)
			failed++
			continue
		}

		// Prefer push remote over fetch remote for remote branch deletion.
		// Guard with RemoteRefExists: the scan phase ran git fetch, so the local
		// tracking refs are current. Skipping branches with no remote counterpart
		// avoids a noisy "remote ref does not exist" error.
		pushRemote := remote
		if c.proj.HasPushRemote && git.RemoteExists(projDir, c.proj.Push) {
			pushRemote = c.proj.Push
		}
		if pushRemote != "" && git.RemoteRefExists(projDir, pushRemote+"/"+c.branch) {
			if err := git.DeleteRemoteBranch(projDir, pushRemote, c.branch); err != nil {
				output.Warning("%s/%s: local deleted, remote delete failed: %s", c.proj.Path, c.branch, err)
			}
		}

		output.Success("%s  %s", c.proj.Path, c.branch)
		deleted++
	}
	return deleted, failed
}
