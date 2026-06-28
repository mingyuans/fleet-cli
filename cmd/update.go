package cmd

import (
	"context"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/selfupdate"
)

var (
	updateCheckOnly bool
	updateForce     bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update fleet to the latest released version",
	Long:  "update checks GitHub Releases for a newer version of fleet and, if found, downloads and installs it in place.",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "only check for a newer version without installing")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "reinstall the latest version even if already up to date")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	output.Info("Current version: %s", version)
	output.Info("Checking for updates...")

	latest, err := selfupdate.LatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("could not fetch the latest version: %w", err)
	}
	output.Info("Latest version:  %s", latest)

	// Decide whether an update is needed unless --force overrides the check.
	if !updateForce {
		if !selfupdate.IsComparable(version) {
			output.Warning("Current build (%s) is a development version and cannot be compared.", version)
			output.Info("Use 'fleet update --force' to install the latest released version.")
			return nil
		}
		cmp, ok := selfupdate.CompareVersions(version, latest)
		if !ok {
			output.Warning("Unable to compare versions; use 'fleet update --force' to reinstall.")
			return nil
		}
		if cmp >= 0 {
			output.Success("Already on the latest version (%s).", version)
			return nil
		}
	}

	if updateCheckOnly {
		output.Info("A new version is available: %s -> %s", version, latest)
		output.Info("Run 'fleet update' to install it.")
		return nil
	}

	return installVersion(ctx, latest)
}

// installVersion downloads, verifies and installs the given release tag.
func installVersion(ctx context.Context, tag string) error {
	assetName, err := selfupdate.AssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	dest, err := selfupdate.CurrentExecutable()
	if err != nil {
		return err
	}

	output.Info("Downloading %s...", assetName)
	asset, checksums, err := selfupdate.DownloadAsset(ctx, tag, assetName)
	if err != nil {
		return err
	}

	if err := selfupdate.VerifyChecksum(asset, assetName, checksums); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}
	output.Success("Checksum verified")

	binary, err := selfupdate.ExtractBinary(asset)
	if err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}

	if err := selfupdate.ReplaceBinary(dest, binary); err != nil {
		return err
	}

	output.Success("Updated fleet to %s", tag)
	output.Info("Installed at %s", dest)
	return nil
}
