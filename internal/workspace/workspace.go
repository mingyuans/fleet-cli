package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xq-yan/fleet-cli/internal/manifest"
)

const (
	defaultManifestFile = "default.xml"
	localManifestFile   = "local_manifest.xml"
)

// Workspace holds the resolved workspace configuration.
type Workspace struct {
	Root             string
	ManifestPath     string
	LocalManifest    string
	HasLocalManifest bool
	Projects         []manifest.ResolvedProject
	SyncJ            int
}

// Load locates manifests, parses, merges, and returns the resolved workspace.
func Load() (*Workspace, error) {
	manifestPath, err := resolveManifestPath()
	if err != nil {
		return nil, err
	}

	root := filepath.Dir(manifestPath)
	localPath := resolveLocalManifestPath(root)

	base, err := manifest.ParseFile(manifestPath)
	if err != nil {
		return nil, err
	}

	ws := &Workspace{
		Root:          root,
		ManifestPath:  manifestPath,
		LocalManifest: localPath,
	}

	merged := base
	if _, statErr := os.Stat(localPath); statErr == nil {
		ws.HasLocalManifest = true
		local, parseErr := manifest.ParseFile(localPath)
		if parseErr != nil {
			return nil, parseErr
		}
		merged = manifest.Merge(base, local)
	}

	projects, syncJ, err := manifest.Resolve(merged)
	if err != nil {
		return nil, err
	}
	ws.Projects = projects
	ws.SyncJ = syncJ

	return ws, nil
}

func resolveManifestPath() (string, error) {
	if env := os.Getenv("FLEET_MANIFEST"); env != "" {
		abs, err := filepath.Abs(env)
		if err != nil {
			return "", fmt.Errorf("resolving FLEET_MANIFEST: %w", err)
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("FLEET_MANIFEST file not found: %s", abs)
		}
		return abs, nil
	}

	return findManifestUpward()
}

func resolveLocalManifestPath(root string) string {
	if env := os.Getenv("FLEET_LOCAL_MANIFEST"); env != "" {
		abs, err := filepath.Abs(env)
		if err == nil {
			return abs
		}
	}
	return filepath.Join(root, localManifestFile)
}

func findManifestUpward() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, defaultManifestFile)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no workspace found (default.xml not found)")
		}
		dir = parent
	}
}
