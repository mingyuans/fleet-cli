package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/executor"
	"github.com/xq-yan/fleet-cli/internal/git"
	"github.com/xq-yan/fleet-cli/internal/manifest"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/workspace"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Clone repositories and configure remotes",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)

	// Print header info
	if ws.Projects[0].HasPushRemote {
		output.Info("Default push remote: %s", output.Bold(ws.Projects[0].Push))
	} else {
		output.Info("fetch-only mode")
	}
	if ws.HasLocalManifest {
		output.Info("Local manifest: %s", ws.LocalManifest)
	}
	if groupFilter != "" {
		output.Info("Group filter: %s", groupFilter)
	}
	output.Header("Initializing %d projects...", len(projects))

	results := executor.Run(projects, ws.SyncJ, func(proj manifest.ResolvedProject, buf *bytes.Buffer, log executor.LogFunc) (string, executor.ResultStatus, string) {
		return initProject(ws.Root, proj, log)
	})

	counts := executor.CountResults(results)
	output.Summary(counts, []string{"cloned", "configured", "skipped", "failed"})

	return nil
}

func initProject(root string, proj manifest.ResolvedProject, log executor.LogFunc) (string, executor.ResultStatus, string) {
	projDir := filepath.Join(root, proj.Path)

	if _, err := os.Stat(projDir); os.IsNotExist(err) {
		return cloneProject(proj, projDir, log)
	}
	return reconcileProject(proj, projDir, log)
}

func cloneProject(proj manifest.ResolvedProject, projDir string, log executor.LogFunc) (string, executor.ResultStatus, string) {
	if err := os.MkdirAll(filepath.Dir(projDir), 0755); err != nil {
		return "failed", executor.StatusFail, err.Error()
	}

	log("cloning from %s ...", proj.Remote)
	if err := git.CloneWithProgress(proj.CloneURL, projDir, proj.Remote, proj.Revision, func(progress string) {
		if strings.Contains(progress, "%") {
			log("%s", progress)
		}
	}); err != nil {
		return "failed", executor.StatusFail, err.Error()
	}

	if proj.HasPushRemote {
		log("adding push remote %s ...", proj.Push)
		if err := git.RemoteAdd(projDir, proj.Push, proj.PushURL); err != nil {
			return "failed", executor.StatusFail, err.Error()
		}
		if err := git.ConfigSet(projDir, "remote.pushDefault", proj.Push); err != nil {
			return "failed", executor.StatusFail, err.Error()
		}
	}

	return "cloned", executor.StatusSuccess, ""
}

func reconcileProject(proj manifest.ResolvedProject, projDir string, log executor.LogFunc) (string, executor.ResultStatus, string) {
	configured := false

	log("checking fetch remote %s ...", proj.Remote)
	if !git.RemoteExists(projDir, proj.Remote) {
		log("adding fetch remote %s ...", proj.Remote)
		if err := git.RemoteAdd(projDir, proj.Remote, proj.CloneURL); err != nil {
			return "failed", executor.StatusFail, err.Error()
		}
		configured = true
	} else {
		url, err := git.RemoteGetURL(projDir, proj.Remote)
		if err != nil {
			return "failed", executor.StatusFail, err.Error()
		}
		if url != proj.CloneURL {
			log("fixing fetch remote URL ...")
			if err := git.RemoteSetURL(projDir, proj.Remote, proj.CloneURL); err != nil {
				return "failed", executor.StatusFail, err.Error()
			}
			configured = true
		}
	}

	if proj.HasPushRemote {
		log("checking push remote %s ...", proj.Push)
		if !git.RemoteExists(projDir, proj.Push) {
			log("adding push remote %s ...", proj.Push)
			if err := git.RemoteAdd(projDir, proj.Push, proj.PushURL); err != nil {
				return "failed", executor.StatusFail, err.Error()
			}
			if err := git.ConfigSet(projDir, "remote.pushDefault", proj.Push); err != nil {
				return "failed", executor.StatusFail, err.Error()
			}
			configured = true
		} else {
			url, err := git.RemoteGetURL(projDir, proj.Push)
			if err != nil {
				return "failed", executor.StatusFail, err.Error()
			}
			if url != proj.PushURL {
				log("fixing push remote URL ...")
				if err := git.RemoteSetURL(projDir, proj.Push, proj.PushURL); err != nil {
					return "failed", executor.StatusFail, err.Error()
				}
				configured = true
			}
		}
	}

	if configured {
		return "configured", executor.StatusSuccess, ""
	}
	return "skipped", executor.StatusSkip, ""
}
