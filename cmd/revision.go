package cmd

import (
	"slices"

	"github.com/xq-yan/fleet-cli/internal/git"
)

// resolveRevision returns the actual branch to use on the remote for a
// configured revision, applying branch-alias fallback. It is a thin wrapper
// around resolveBranchWithAliases.
func resolveRevision(dir, remote, revision string, groups [][]string) string {
	return resolveBranchWithAliases(dir, remote, revision, groups)
}

// resolveBranchWithAliases returns the branch actually available on the remote.
// It first tries branch itself; if that ref is absent and branch belongs to an
// alias group, it falls back to the first existing member of that group. The
// requested branch is always tried first; the remaining members are tried in
// their declared order as a deterministic tie-break. Returns "" when nothing
// usable exists.
func resolveBranchWithAliases(dir, remote, branch string, groups [][]string) string {
	if git.RemoteRefExists(dir, remote+"/"+branch) {
		return branch
	}
	for _, candidate := range branchAliasGroup(branch, groups) {
		if candidate == branch {
			continue
		}
		if git.RemoteRefExists(dir, remote+"/"+candidate) {
			return candidate
		}
	}
	return ""
}

// branchAliasGroup returns the first alias group that contains branch, or nil
// when branch belongs to no group.
func branchAliasGroup(branch string, groups [][]string) []string {
	for _, g := range groups {
		if slices.Contains(g, branch) {
			return g
		}
	}
	return nil
}

// resolveRemote returns the remote name to use, falling back to "origin".
func resolveRemote(dir, preferred string) string {
	if git.RemoteExists(dir, preferred) {
		return preferred
	}
	if git.RemoteExists(dir, "origin") {
		return "origin"
	}
	return ""
}
