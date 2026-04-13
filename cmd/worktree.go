package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/executor"
	"github.com/xq-yan/fleet-cli/internal/git"
	"github.com/xq-yan/fleet-cli/internal/manifest"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/workspace"
)

var worktreeBranch string
var worktreeRevision string
var worktreeDest string

var worktreeCmd = &cobra.Command{
	Use:   "worktree <name>",
	Short: "Create a git worktree across all repositories",
	Long: `Create a git worktree for each managed repository under <worktree-base>/<name>/.

The worktree directory mirrors the original workspace structure. If the branch
does not exist locally it is created from <remote>/<revision>.

Use --dest to place worktrees at an explicit directory, bypassing worktree-base.
When --dest is used, <name> can be omitted but --branch is required:
  fleet worktree --dest ~/worktrees/feature-x -b feature-x

Configure the default base path in fleet.xml:
  <default worktree-base="~/worktrees/myproject" worktree-copy=".env,.env.*" />`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runWorktree,
}

func init() {
	worktreeCmd.Flags().StringVarP(&worktreeBranch, "branch", "b", "",
		"branch name to create or checkout (default: worktree name)")
	worktreeCmd.Flags().StringVarP(&worktreeRevision, "revision", "r", "",
		"upstream revision to base the new branch on (default: project revision in fleet.xml)")
	worktreeCmd.Flags().StringVarP(&worktreeDest, "dest", "d", "",
		"destination directory for worktrees (overrides worktree-base/<name>)")
	rootCmd.AddCommand(worktreeCmd)
}

func runWorktree(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	if worktreeDest == "" && name == "" {
		return fmt.Errorf("requires <name> argument (or use --dest with --branch)")
	}
	if worktreeDest != "" && worktreeBranch == "" {
		return fmt.Errorf("--branch is required when using --dest")
	}

	branch := worktreeBranch
	if branch == "" {
		branch = name
	}

	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	var worktreeRoot string
	switch {
	case worktreeDest != "":
		worktreeRoot = workspace.ExpandHome(worktreeDest)
	case ws.WorktreeBase != "":
		worktreeRoot = filepath.Join(ws.WorktreeBase, name)
	default:
		return fmt.Errorf(`worktree-base is not configured; add it to <default> in fleet.xml or use --dest:
  <default worktree-base="~/worktrees/myproject" />
  fleet worktree --dest ~/worktrees/feature-x -b feature-x`)
	}

	projects := filterByGroup(ws.Projects)
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}

	output.Header("Creating worktree %s across %d projects...", output.Bold(branch), len(projects))

	// Split projects: root (path=".") must run and complete before services start
	// creating parent directories, otherwise MkdirAll creates the worktree root
	// directory and causes git worktree add to fail with "already exists".
	var rootProjs, otherProjs []manifest.ResolvedProject
	for _, p := range projects {
		if filepath.Clean(p.Path) == "." {
			rootProjs = append(rootProjs, p)
		} else {
			otherProjs = append(otherProjs, p)
		}
	}

	fn := func(proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
		return worktreeProject(ws.Root, worktreeRoot, proj, branch, worktreeRevision, log)
	}

	total := len(projects)
	var allResults []executor.Result

	// Phase 1: root project first (sequential), so git creates the worktree root dir.
	if len(rootProjs) > 0 {
		allResults = append(allResults, executor.RunWithOffset(rootProjs, len(rootProjs), 0, total, fn)...)
	}
	// Phase 2: remaining projects in parallel, safe to MkdirAll inside worktree root.
	if len(otherProjs) > 0 {
		allResults = append(allResults, executor.RunWithOffset(otherProjs, ws.SyncJ, len(rootProjs), total, fn)...)
	}

	counts := executor.CountResults(allResults)
	output.Summary(counts, []string{"created", "skipped", "failed"})
	return nil
}

func worktreeProject(root, worktreeRoot string, proj manifest.ResolvedProject, branch, revision string, log executor.LogFunc) (string, executor.ResultStatus, string) {
	projDir := filepath.Join(root, proj.Path)

	if _, err := os.Stat(projDir); os.IsNotExist(err) {
		return "skipped", executor.StatusSkip, "not cloned"
	}

	wtPath := filepath.Join(worktreeRoot, proj.Path)

	// Checking the directory alone is unreliable: parent dirs may have been
	// created by sibling projects running in parallel via MkdirAll.
	if _, err := os.Stat(filepath.Join(wtPath, ".git")); err == nil {
		return "skipped", executor.StatusSkip, "worktree already exists"
	}

	if err := os.MkdirAll(filepath.Dir(wtPath), 0o755); err != nil {
		return "failed", executor.StatusFail, "mkdir: " + err.Error()
	}

	// Determine effective revision: flag > project field.
	effectiveRevision := revision
	if effectiveRevision == "" {
		effectiveRevision = proj.Revision
	}

	if git.BranchExists(projDir, branch) {
		log("adding worktree %s (branch %s) ...", wtPath, branch)
		if err := git.WorktreeAdd(projDir, wtPath, branch); err != nil {
			return "failed", executor.StatusFail, err.Error()
		}
	} else {
		remote := resolveRemote(projDir, proj.Remote)
		if remote == "" {
			return "failed", executor.StatusFail, "no suitable remote found"
		}
		rev := resolveRevision(projDir, remote, effectiveRevision, proj.MasterMainCompat)
		if rev == "" {
			rev = effectiveRevision
		}
		startPoint := remote + "/" + rev
		log("adding worktree %s (new branch %s from %s) ...", wtPath, branch, startPoint)
		if err := git.WorktreeAddNew(projDir, wtPath, branch, startPoint); err != nil {
			return "failed", executor.StatusFail, err.Error()
		}
	}

	// Copy gitignored files specified by worktree-copy patterns.
	for _, pattern := range proj.WorktreeCopy {
		matches, err := filepath.Glob(filepath.Join(projDir, pattern))
		if err != nil || len(matches) == 0 {
			continue
		}
		for _, src := range matches {
			rel, err := filepath.Rel(projDir, src)
			if err != nil {
				continue
			}
			dst := filepath.Join(wtPath, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				continue
			}
			if err := copyFile(src, dst); err != nil {
				log("warning: copy %s: %v", rel, err)
			}
		}
	}

	return "created", executor.StatusSuccess, ""
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
