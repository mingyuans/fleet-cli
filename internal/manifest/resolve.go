package manifest

import (
	"fmt"
	"strconv"
	"strings"
)

const defaultSyncJ = 4

// Resolve takes a merged manifest and returns resolved projects, sync concurrency,
// worktree base path, and any error.
func Resolve(m *Manifest) ([]ResolvedProject, int, string, error) {
	remoteMap := make(map[string]Remote, len(m.Remotes))
	for _, r := range m.Remotes {
		remoteMap[r.Name] = r
	}

	syncJ := defaultSyncJ
	var defaultRemote, defaultRevision, defaultPush, defaultWorktreeBase, defaultWorktreeCopy string
	var masterMainCompat bool

	if m.Default != nil {
		defaultRemote = m.Default.Remote
		defaultRevision = m.Default.Revision
		defaultPush = m.Default.Push
		defaultWorktreeBase = m.Default.WorktreeBase
		defaultWorktreeCopy = m.Default.WorktreeCopy
		masterMainCompat = m.Default.MasterMainCompat == "true"
		if m.Default.SyncJ != "" {
			if v, err := strconv.Atoi(m.Default.SyncJ); err == nil && v > 0 {
				syncJ = v
			}
		}
	}

	aliasGroups := normalizeAliasGroups(m.BranchAliases, masterMainCompat)

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
			return nil, 0, "", fmt.Errorf("project %q references unknown remote %q", p.Name, rp.Remote)
		}
		rp.CloneURL = ensureTrailingSlash(fetchRemote.Fetch) + p.Name + ".git"

		rp.AliasGroups = aliasGroups

		if rp.Push != "" {
			rp.HasPushRemote = true
			if rp.Push != rp.Remote {
				pushRemote, ok := remoteMap[rp.Push]
				if !ok {
					return nil, 0, "", fmt.Errorf("project %q references unknown push remote %q", p.Name, rp.Push)
				}
				rp.PushURL = ensureTrailingSlash(pushRemote.Fetch) + p.Name + ".git"
			}
		}

		copyStr := p.WorktreeCopy
		if copyStr == "" {
			copyStr = defaultWorktreeCopy
		}
		if copyStr != "" {
			for _, pat := range strings.Split(copyStr, ",") {
				if pat = strings.TrimSpace(pat); pat != "" {
					rp.WorktreeCopy = append(rp.WorktreeCopy, pat)
				}
			}
		}

		resolved = append(resolved, rp)
	}

	return resolved, syncJ, defaultWorktreeBase, nil
}

func ensureTrailingSlash(s string) string {
	if s != "" && s[len(s)-1] != '/' {
		return s + "/"
	}
	return s
}

// normalizeAliasGroups turns the raw <branch-alias> elements into clean alias
// groups: each member is trimmed, blank members are dropped, intra-group
// duplicates are removed, and groups with fewer than 2 valid members are
// ignored. When masterMainCompat is enabled and no explicit group already
// covers master/main, a built-in ["master", "main"] group is appended so the
// legacy flag keeps working.
func normalizeAliasGroups(aliases []BranchAlias, masterMainCompat bool) [][]string {
	var groups [][]string
	for _, a := range aliases {
		seen := make(map[string]bool, len(a.Branches))
		var members []string
		for _, b := range a.Branches {
			b = strings.TrimSpace(b)
			if b == "" || seen[b] {
				continue
			}
			seen[b] = true
			members = append(members, b)
		}
		if len(members) >= 2 {
			groups = append(groups, members)
		}
	}

	if masterMainCompat && !groupsCoverMasterMain(groups) {
		groups = append(groups, []string{"master", "main"})
	}
	return groups
}

// groupsCoverMasterMain reports whether any alias group already contains
// "master" or "main".
func groupsCoverMasterMain(groups [][]string) bool {
	for _, g := range groups {
		for _, b := range g {
			if b == "master" || b == "main" {
				return true
			}
		}
	}
	return false
}
