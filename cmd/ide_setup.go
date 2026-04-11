package cmd

import "github.com/spf13/cobra"

var ideSetupCmd = &cobra.Command{
	Use:   "ide-setup",
	Short: "Configure IDE settings for the workspace",
}

func init() {
	rootCmd.AddCommand(ideSetupCmd)
}
