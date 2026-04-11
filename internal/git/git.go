package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Clone clones a repository.
func Clone(url, path, origin, branch string) error {
	return run(".", "git", "clone", url, path, "--origin", origin, "-b", branch)
}

// CloneWithProgress clones a repository and reports stderr progress via onProgress.
// Each call to onProgress receives a trimmed progress line like "Receiving objects: 45% (123/273)".
func CloneWithProgress(url, path, origin, branch string, onProgress func(string)) error {
	cmd := exec.Command("git", "clone", "--progress", url, path, "--origin", origin, "-b", branch)
	cmd.Dir = "."

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting git clone: %w", err)
	}

	// Git progress uses \r to overwrite lines, read byte-by-byte and split on \r or \n.
	buf := make([]byte, 0, 256)
	tmp := make([]byte, 1)
	for {
		n, readErr := stderr.Read(tmp)
		if n > 0 {
			ch := tmp[0]
			if ch == '\r' || ch == '\n' {
				line := strings.TrimSpace(string(buf))
				if line != "" && onProgress != nil {
					onProgress(line)
				}
				buf = buf[:0]
			} else {
				buf = append(buf, ch)
			}
		}
		if readErr != nil {
			break
		}
	}
	// Flush remaining
	if line := strings.TrimSpace(string(buf)); line != "" && onProgress != nil {
		onProgress(line)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	return nil
}

// RemoteAdd adds a new remote.
func RemoteAdd(dir, name, url string) error {
	return run(dir, "git", "remote", "add", name, url)
}

// RemoteSetURL sets the URL for an existing remote.
func RemoteSetURL(dir, name, url string) error {
	return run(dir, "git", "remote", "set-url", name, url)
}

// RemoteGetURL returns the fetch URL for a remote.
func RemoteGetURL(dir, name string) (string, error) {
	return output(dir, "git", "remote", "get-url", name)
}

// ConfigSet sets a git config value.
func ConfigSet(dir, key, value string) error {
	return run(dir, "git", "config", key, value)
}

// Fetch fetches from a remote.
func Fetch(dir, remote string) error {
	return run(dir, "git", "fetch", remote)
}

// PullRebase performs a pull with rebase from a remote branch.
func PullRebase(dir, remote, branch string) error {
	return run(dir, "git", "pull", "--rebase", remote, branch)
}

// Push pushes a branch to a remote.
func Push(dir, remote, branch string) error {
	return run(dir, "git", "push", remote, branch)
}

// CurrentBranch returns the current branch name.
// Returns empty string and no error for detached HEAD.
func CurrentBranch(dir string) (string, error) {
	out, err := output(dir, "git", "symbolic-ref", "--short", "HEAD")
	if err != nil {
		// Detached HEAD results in a non-zero exit
		return "", nil
	}
	return out, nil
}

// StatusPorcelain returns the porcelain status output.
func StatusPorcelain(dir string) (string, error) {
	return output(dir, "git", "status", "--porcelain")
}

// AheadBehind returns the ahead/behind counts relative to a remote tracking ref.
func AheadBehind(dir, remote, branch string) (ahead, behind int, err error) {
	ref := remote + "/" + branch
	out, err := output(dir, "git", "rev-list", "--left-right", "--count", "HEAD..."+ref)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Fields(out)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", out)
	}
	ahead, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing ahead count: %w", err)
	}
	behind, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing behind count: %w", err)
	}
	return ahead, behind, nil
}

// BranchExists checks if a local branch exists.
func BranchExists(dir, branch string) bool {
	_, err := output(dir, "git", "rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}

// CheckoutBranch switches to an existing local branch.
func CheckoutBranch(dir, branch string) error {
	return run(dir, "git", "checkout", branch)
}

// CreateBranchFrom creates a new branch from a start point and checks it out.
func CreateBranchFrom(dir, branch, startPoint string) error {
	return run(dir, "git", "checkout", "-b", branch, startPoint)
}

// DeleteBranch deletes a local branch (-D force delete).
func DeleteBranch(dir, branch string) error {
	return run(dir, "git", "branch", "-D", branch)
}

// DeleteRemoteBranch deletes a branch on a remote.
func DeleteRemoteBranch(dir, remote, branch string) error {
	return run(dir, "git", "push", remote, "--delete", branch)
}

// RemoteRefExists checks if a remote ref (e.g. origin/main) exists.
func RemoteRefExists(dir, ref string) bool {
	_, err := output(dir, "git", "rev-parse", "--verify", "refs/remotes/"+ref)
	return err == nil
}

// RemoteExists checks if a remote exists in the repository.
func RemoteExists(dir, name string) bool {
	_, err := output(dir, "git", "remote", "get-url", name)
	return err == nil
}

// ParseRepoOwner extracts "owner/repo" from a git remote URL.
// Supports SSH (git@github.com:owner/repo.git) and HTTPS (https://github.com/owner/repo.git).
func ParseRepoOwner(url string) (host, ownerRepo string, ok bool) {
	// SSH: git@github.com:owner/repo.git
	if strings.Contains(url, "@") && strings.Contains(url, ":") {
		atIdx := strings.Index(url, "@")
		colonIdx := strings.Index(url[atIdx:], ":") + atIdx
		host = url[atIdx+1 : colonIdx]
		ownerRepo = strings.TrimSuffix(url[colonIdx+1:], ".git")
		return host, ownerRepo, ownerRepo != ""
	}
	// HTTPS: https://github.com/owner/repo.git
	if strings.Contains(url, "://") {
		url = strings.TrimSuffix(url, ".git")
		parts := strings.SplitN(url, "://", 2)
		if len(parts) != 2 {
			return "", "", false
		}
		segments := strings.SplitN(parts[1], "/", 2)
		if len(segments) != 2 {
			return "", "", false
		}
		host = segments[0]
		ownerRepo = segments[1]
		return host, ownerRepo, ownerRepo != ""
	}
	return "", "", false
}

func run(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return fmt.Errorf("%s: %s", err, errMsg)
		}
		return err
	}
	return nil
}

func output(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("%s: %s", err, errMsg)
		}
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}
