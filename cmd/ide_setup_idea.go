package cmd

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xq-yan/fleet-cli/internal/output"
	"github.com/xq-yan/fleet-cli/internal/workspace"
)

type vcsProject struct {
	XMLName xml.Name     `xml:"project"`
	Version string       `xml:"version,attr"`
	Comp    vcsComponent `xml:"component"`
}

type vcsComponent struct {
	Name     string       `xml:"name,attr"`
	Mappings []vcsMapping `xml:"mapping"`
}

type vcsMapping struct {
	Directory string `xml:"directory,attr"`
	VCS       string `xml:"vcs,attr"`
}

var ideSetupIdeaCmd = &cobra.Command{
	Use:   "idea",
	Short: "Configure IntelliJ IDEA / GoLand VCS directory mappings",
	RunE:  runIdeSetupIdea,
}

func init() {
	ideSetupCmd.AddCommand(ideSetupIdeaCmd)
}

func runIdeSetupIdea(cmd *cobra.Command, args []string) error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	projects := filterByGroup(ws.Projects)

	ideaDir := filepath.Join(ws.Root, ".idea")
	if _, err := os.Stat(ideaDir); os.IsNotExist(err) {
		if err := os.MkdirAll(ideaDir, 0755); err != nil {
			return fmt.Errorf("creating .idea directory: %w", err)
		}
	}

	// Build mappings: root project + each sub-project
	var mappings []vcsMapping
	mappings = append(mappings, vcsMapping{Directory: "$PROJECT_DIR$", VCS: "Git"})

	for _, proj := range projects {
		dir := "$PROJECT_DIR$/" + proj.Path
		if _, err := os.Stat(filepath.Join(ws.Root, proj.Path)); os.IsNotExist(err) {
			continue
		}
		mappings = append(mappings, vcsMapping{Directory: dir, VCS: "Git"})
	}

	doc := vcsProject{
		Version: "4",
		Comp: vcsComponent{
			Name:     "VcsDirectoryMappings",
			Mappings: mappings,
		},
	}

	vcsPath := filepath.Join(ideaDir, "vcs.xml")
	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling vcs.xml: %w", err)
	}

	content := xml.Header + string(data) + "\n"
	if err := os.WriteFile(vcsPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing vcs.xml: %w", err)
	}

	output.Success("Wrote %s with %d VCS mappings", vcsPath, len(mappings))
	return nil
}
