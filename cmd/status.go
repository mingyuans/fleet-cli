package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/git"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/workspace"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all repositories",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}

	table := output.NewTable("PROJECT", "BRANCH", "STATUS", "AHEAD/BEHIND", "FETCH", "PUSH")

	for _, proj := range projects {
		projDir := filepath.Join(ws.Root, proj.Path)

		fetch := proj.Remote + "/" + proj.Revision
		push := "–"
		if proj.HasPushRemote {
			push = proj.Push
		}

		if _, err := os.Stat(projDir); os.IsNotExist(err) {
			table.AddRow(
				[]string{proj.Name, "–", "not cloned", "", fetch, push},
				[]string{output.ColorGrey, output.ColorGrey, output.ColorGrey, output.ColorGrey, output.ColorGrey, output.ColorGrey},
			)
			continue
		}

		branch, err := git.CurrentBranch(projDir)
		if err != nil {
			table.AddRow([]string{proj.Name, "?", "error", err.Error(), fetch, push}, nil)
			continue
		}
		if branch == "" {
			branch = "(detached)"
		}

		status, err := git.StatusPorcelain(projDir)
		if err != nil {
			table.AddRow([]string{proj.Name, branch, "error", err.Error(), fetch, push}, nil)
			continue
		}

		statusText := "clean"
		var statusColor string
		if status != "" {
			statusText = "dirty"
			statusColor = output.ColorYellow
		} else {
			statusColor = output.ColorGreen
		}

		var branchColor string
		if branch != proj.Revision && branch != "(detached)" {
			branchColor = output.ColorYellow
		}

		// Ahead/behind
		aheadBehind := ""
		remote := resolveRemote(projDir, proj.Remote)
		if remote != "" && branch != "(detached)" {
			ahead, behind, abErr := git.AheadBehind(projDir, remote, proj.Revision)
			if abErr == nil && (ahead > 0 || behind > 0) {
				aheadBehind = fmt.Sprintf("+%d -%d", ahead, behind)
			}
		}

		table.AddRow(
			[]string{proj.Name, branch, statusText, aheadBehind, fetch, push},
			[]string{"", branchColor, statusColor, "", "", ""},
		)
	}

	fmt.Println()
	table.Print()

	return nil
}
