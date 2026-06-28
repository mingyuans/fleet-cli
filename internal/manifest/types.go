package manifest

import "encoding/xml"

// Manifest represents the root <manifest> element.
type Manifest struct {
	XMLName       xml.Name      `xml:"manifest"`
	Remotes       []Remote      `xml:"remote"`
	Default       *Default      `xml:"default"`
	BranchAliases []BranchAlias `xml:"branch-alias"`
	Projects      []Project     `xml:"project"`
}

// BranchAlias represents a <branch-alias> element: a group of branches that are
// treated as peer equivalents (no master/slave). Members are listed as <branch>
// child elements.
type BranchAlias struct {
	Branches []string `xml:"branch"`
}

// Remote represents a <remote> element.
type Remote struct {
	Name   string `xml:"name,attr"`
	Fetch  string `xml:"fetch,attr"`
	Review string `xml:"review,attr,omitempty"`
}

// Default represents the <default> element.
type Default struct {
	Remote           string `xml:"remote,attr,omitempty"`
	Revision         string `xml:"revision,attr,omitempty"`
	SyncJ            string `xml:"sync-j,attr,omitempty"`
	Push             string `xml:"push,attr,omitempty"`
	MasterMainCompat string `xml:"master-main-compat,attr,omitempty"`
	WorktreeBase     string `xml:"worktree-base,attr,omitempty"`
	WorktreeCopy     string `xml:"worktree-copy,attr,omitempty"`
}

// Project represents a <project> element.
type Project struct {
	Name         string `xml:"name,attr"`
	Path         string `xml:"path,attr"`
	Groups       string `xml:"groups,attr,omitempty"`
	Remote       string `xml:"remote,attr,omitempty"`
	Revision     string `xml:"revision,attr,omitempty"`
	Push         string `xml:"push,attr,omitempty"`
	WorktreeCopy string `xml:"worktree-copy,attr,omitempty"`
}

// ResolvedProject holds a project with its effective (resolved) configuration.
type ResolvedProject struct {
	Name          string
	Path          string
	Groups        []string
	Remote        string
	Revision      string
	Push          string
	CloneURL      string
	PushURL       string
	HasPushRemote bool
	AliasGroups   [][]string // normalized branch alias groups (peer-equivalent members)
	WorktreeCopy  []string   // glob patterns for files to copy into new worktrees
}

// IsForkPush reports whether the push remote differs from the fetch remote,
// indicating a fork-based workflow that requires a separate git remote.
func (rp ResolvedProject) IsForkPush() bool {
	return rp.HasPushRemote && rp.Push != rp.Remote
}
