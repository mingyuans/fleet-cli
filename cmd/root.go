package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var groupFilter string

var rootCmd = &cobra.Command{
	Use:   "fleet",
	Short: "Multi-repo workspace management tool",
	Long:  "fleet manages multiple Git repositories in a workspace using manifest files.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&groupFilter, "group", "g", "", "filter projects by group")
}
