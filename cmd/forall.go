package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/workspace"
)

var forallShellCmd string

var forallCmd = &cobra.Command{
	Use:   "forall [-- command args...]",
	Short: "Execute a command in all repositories",
	RunE:  runForall,
}

func init() {
	forallCmd.Flags().StringVarP(&forallShellCmd, "command", "c", "", "shell command to execute")
	rootCmd.AddCommand(forallCmd)
}

func runForall(cmd *cobra.Command, args []string) error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}

	if forallShellCmd == "" && len(args) == 0 {
		return fmt.Errorf("either -c <command> or -- <command> [args...] is required")
	}

	for _, proj := range projects {
		projDir := filepath.Join(ws.Root, proj.Path)

		// Skip uncloned
		if _, err := os.Stat(projDir); os.IsNotExist(err) {
			continue
		}

		// Print project header
		fmt.Printf("\n%s %s (%s)\n", output.Bold(output.IconInfo), output.Bold(proj.Name), proj.Path)

		var c *exec.Cmd
		if forallShellCmd != "" {
			c = exec.Command("sh", "-c", forallShellCmd)
		} else {
			c = exec.Command(args[0], args[1:]...)
		}
		c.Dir = projDir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Env = append(os.Environ(),
			"FLEET_PROJECT_NAME="+proj.Name,
			"FLEET_PROJECT_PATH="+proj.Path,
			"FLEET_PROJECT_GROUPS="+strings.Join(proj.Groups, ","),
			"FLEET_PROJECT_REMOTE="+proj.Remote,
			"FLEET_PROJECT_REVISION="+proj.Revision,
			"FLEET_PROJECT_PUSH_REMOTE="+proj.Push,
		)

		if err := c.Run(); err != nil {
			output.Warning("%s: command failed: %v", proj.Name, err)
		}
	}

	return nil
}
