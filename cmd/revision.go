package cmd

import "github.com/xq-yan/fleet-cli/internal/git"

// resolveRevision returns the actual branch to use on the remote.
// If the configured revision exists, it is returned directly.
// When masterMainCompat is true and revision is "master" or "main",
// it falls back to the peer branch if the configured one doesn't exist.
func resolveRevision(dir, remote, revision string, masterMainCompat bool) string {
	ref := remote + "/" + revision
	if git.RemoteRefExists(dir, ref) {
		return revision
	}
	if masterMainCompat {
		peer := masterMainPeer(revision)
		if peer != "" && git.RemoteRefExists(dir, remote+"/"+peer) {
			return peer
		}
	}
	return ""
}

// masterMainPeer returns the counterpart of master/main. Empty if neither.
func masterMainPeer(branch string) string {
	switch branch {
	case "master":
		return "main"
	case "main":
		return "master"
	default:
		return ""
	}
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
