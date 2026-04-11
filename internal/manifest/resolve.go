package manifest

import (
	"fmt"
	"strconv"
	"strings"
)

const defaultSyncJ = 4

// Resolve takes a merged manifest and returns resolved projects with effective configuration.
func Resolve(m *Manifest) ([]ResolvedProject, int, error) {
	remoteMap := make(map[string]Remote, len(m.Remotes))
	for _, r := range m.Remotes {
		remoteMap[r.Name] = r
	}

	syncJ := defaultSyncJ
	var defaultRemote, defaultRevision, defaultPush string
	var masterMainCompat bool

	if m.Default != nil {
		defaultRemote = m.Default.Remote
		defaultRevision = m.Default.Revision
		defaultPush = m.Default.Push
		masterMainCompat = m.Default.MasterMainCompat == "true"
		if m.Default.SyncJ != "" {
			if v, err := strconv.Atoi(m.Default.SyncJ); err == nil && v > 0 {
				syncJ = v
			}
		}
	}

	resolved := make([]ResolvedProject, 0, len(m.Projects))
	for _, p := range m.Projects {
		rp := ResolvedProject{
			Name: p.Name,
			Path: p.Path,
		}

		if p.Groups != "" {
			rp.Groups = strings.Split(p.Groups, ",")
		}

		rp.Remote = p.Remote
		if rp.Remote == "" {
			rp.Remote = defaultRemote
		}

		rp.Revision = p.Revision
		if rp.Revision == "" {
			rp.Revision = defaultRevision
		}

		rp.Push = p.Push
		if rp.Push == "" {
			rp.Push = defaultPush
		}

		fetchRemote, ok := remoteMap[rp.Remote]
		if !ok {
			return nil, 0, fmt.Errorf("project %q references unknown remote %q", p.Name, rp.Remote)
		}
		rp.CloneURL = fetchRemote.Fetch + p.Name + ".git"

		rp.MasterMainCompat = masterMainCompat

		if rp.Push != "" && rp.Push != rp.Remote {
			pushRemote, ok := remoteMap[rp.Push]
			if !ok {
				return nil, 0, fmt.Errorf("project %q references unknown push remote %q", p.Name, rp.Push)
			}
			rp.PushURL = pushRemote.Fetch + p.Name + ".git"
			rp.HasPushRemote = true
		}

		resolved = append(resolved, rp)
	}

	return resolved, syncJ, nil
}
