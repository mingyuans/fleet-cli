package manifest

// Merge merges a local manifest into a base manifest.
// Rules:
// - Remotes: same-name remotes are replaced, new remotes are appended.
// - Default: per-attribute override (non-empty local attrs overwrite base).
// - Projects: same-name projects use per-attribute override; new projects are appended.
func Merge(base, local *Manifest) *Manifest {
	result := &Manifest{}
	result.Remotes = mergeRemotes(base.Remotes, local.Remotes)
	result.Default = mergeDefault(base.Default, local.Default)
	result.Projects = mergeProjects(base.Projects, local.Projects)
	return result
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
