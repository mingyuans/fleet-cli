package manifest

import (
	"sort"
	"strings"
)

// Merge merges a local manifest into a base manifest.
// Rules:
// - Remotes: same-name remotes are replaced, new remotes are appended.
// - Default: per-attribute override (non-empty local attrs overwrite base).
// - BranchAliases: same-member-set groups are replaced, new groups are appended.
// - Projects: same-name projects use per-attribute override; new projects are appended.
func Merge(base, local *Manifest) *Manifest {
	result := &Manifest{}
	result.Remotes = mergeRemotes(base.Remotes, local.Remotes)
	result.Default = mergeDefault(base.Default, local.Default)
	result.BranchAliases = mergeBranchAliases(base.BranchAliases, local.BranchAliases)
	result.Projects = mergeProjects(base.Projects, local.Projects)
	return result
}

// mergeBranchAliases merges alias groups by their member set: a local group
// whose normalized members match a base group replaces it, otherwise it is
// appended. This keeps merging deterministic regardless of member order.
func mergeBranchAliases(base, local []BranchAlias) []BranchAlias {
	index := make(map[string]int, len(base))
	result := make([]BranchAlias, len(base))
	copy(result, base)

	for i, a := range result {
		index[aliasKey(a)] = i
	}

	for _, la := range local {
		key := aliasKey(la)
		if idx, ok := index[key]; ok {
			result[idx] = la
		} else {
			index[key] = len(result)
			result = append(result, la)
		}
	}
	return result
}

// aliasKey returns an order-independent key for a branch alias group, built from
// its trimmed, sorted members.
func aliasKey(a BranchAlias) string {
	members := make([]string, 0, len(a.Branches))
	for _, b := range a.Branches {
		if b = strings.TrimSpace(b); b != "" {
			members = append(members, b)
		}
	}
	sort.Strings(members)
	return strings.Join(members, "\x00")
}

func mergeRemotes(base, local []Remote) []Remote {
	index := make(map[string]int, len(base))
	result := make([]Remote, len(base))
	copy(result, base)

	for i, r := range result {
		index[r.Name] = i
	}

	for _, lr := range local {
		if idx, ok := index[lr.Name]; ok {
			result[idx] = lr
		} else {
			result = append(result, lr)
		}
	}
	return result
}

func mergeDefault(base, local *Default) *Default {
	if base == nil && local == nil {
		return nil
	}
	result := &Default{}
	if base != nil {
		*result = *base
	}
	if local == nil {
		return result
	}
	if local.Remote != "" {
		result.Remote = local.Remote
	}
	if local.Revision != "" {
		result.Revision = local.Revision
	}
	if local.SyncJ != "" {
		result.SyncJ = local.SyncJ
	}
	if local.Push != "" {
		result.Push = local.Push
	}
	if local.MasterMainCompat != "" {
		result.MasterMainCompat = local.MasterMainCompat
	}
	return result
}

func mergeProjects(base, local []Project) []Project {
	index := make(map[string]int, len(base))
	result := make([]Project, len(base))
	copy(result, base)

	for i, p := range result {
		index[p.Name] = i
	}

	for _, lp := range local {
		if idx, ok := index[lp.Name]; ok {
			bp := &result[idx]
			if lp.Path != "" {
				bp.Path = lp.Path
			}
			if lp.Groups != "" {
				bp.Groups = lp.Groups
			}
			if lp.Remote != "" {
				bp.Remote = lp.Remote
			}
			if lp.Revision != "" {
				bp.Revision = lp.Revision
			}
			if lp.Push != "" {
				bp.Push = lp.Push
			}
		} else {
			result = append(result, lp)
		}
	}
	return result
}
