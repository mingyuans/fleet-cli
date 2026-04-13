package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
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

	candidates := collectPruneCandidates(ws.Root, projects)

	if len(candidates) == 0 {
		fmt.Println()
		output.Info("No merged branches found.")
		return nil
	}

	// Display candidates.
	fmt.Println()
	output.Info("The following merged branches can be safely deleted:")
	fmt.Println()

	table := output.NewTable("PROJECT", "BRANCH")
	for _, c := range candidates {
		table.AddRow([]string{c.proj.Path, c.branch}, []string{"", output.ColorYellow})
	}
	table.Print()

	// Confirm.
	fmt.Println()
	fmt.Printf("Delete %d branch(es) (local + remote)? [y/N] ", len(candidates))

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if answer := strings.TrimSpace(scanner.Text()); !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
		fmt.Println()
		output.Info("Aborted.")
		return nil
	}

	// Delete.
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

// collectPruneCandidates fetches each project and returns branches merged into its revision.
func collectPruneCandidates(root string, projects []manifest.ResolvedProject) []pruneCandidate {
	var candidates []pruneCandidate

	for _, proj := range projects {
		projDir := filepath.Join(root, proj.Path)

		if _, err := os.Stat(projDir); os.IsNotExist(err) {
			output.Skip("%s (not cloned)", proj.Path)
			continue
		}

		remote := resolveRemote(projDir, proj.Remote)
		if remote == "" {
			output.Warning("%s: no suitable remote found (tried %s and origin)", proj.Path, proj.Remote)
			continue
		}

		if err := git.Fetch(projDir, remote); err != nil {
			output.Warning("%s: fetch failed: %s", proj.Path, err)
			continue
		}

		revision := resolveRevision(projDir, remote, proj.Revision, proj.MasterMainCompat)
		if revision == "" {
			output.Warning("%s: revision %s not found on remote", proj.Path, proj.Revision)
			continue
		}

		mergeBase := remote + "/" + revision
		branches, err := git.ListMergedBranches(projDir, mergeBase)
		if err != nil {
			output.Warning("%s: cannot list merged branches: %s", proj.Path, err)
			continue
		}

		for _, branch := range branches {
			if slices.Contains(protectedBranches, branch) {
				continue
			}
			if branch == revision {
				continue
			}
			candidates = append(candidates, pruneCandidate{proj: proj, branch: branch})
		}
	}

	return candidates
}

// deleteCandidates deletes each candidate branch locally and on the remote.
// Returns counts of deleted and failed branches.
func deleteCandidates(root string, candidates []pruneCandidate) (deleted, failed int) {
	for _, c := range candidates {
		projDir := filepath.Join(root, c.proj.Path)

		// If currently on the branch to be deleted, switch away first.
		current, err := git.CurrentBranch(projDir)
		if err != nil {
			output.Error("%s/%s: cannot determine current branch: %s", c.proj.Path, c.branch, err)
			failed++
			continue
		}
		if current == c.branch {
			remote := resolveRemote(projDir, c.proj.Remote)
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

		// Delete remote branch. Prefer push remote if available.
		pushRemote := resolveRemote(projDir, c.proj.Remote)
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
