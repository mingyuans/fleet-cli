package manifest

import "encoding/xml"

// Manifest represents the root <manifest> element.
type Manifest struct {
	XMLName  xml.Name  `xml:"manifest"`
	Remotes  []Remote  `xml:"remote"`
	Default  *Default  `xml:"default"`
	Projects []Project `xml:"project"`
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
}

// Project represents a <project> element.
type Project struct {
	Name     string `xml:"name,attr"`
	Path     string `xml:"path,attr"`
	Groups   string `xml:"groups,attr,omitempty"`
	Remote   string `xml:"remote,attr,omitempty"`
	Revision string `xml:"revision,attr,omitempty"`
	Push     string `xml:"push,attr,omitempty"`
}

// ResolvedProject holds a project with its effective (resolved) configuration.
type ResolvedProject struct {
	Name             string
	Path             string
	Groups           []string
	Remote           string
	Revision         string
	Push             string
	CloneURL         string
	PushURL          string
	HasPushRemote    bool
	MasterMainCompat bool
}
