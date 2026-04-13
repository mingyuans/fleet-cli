package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

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

	// Git progress uses \r to overwrite lines; use buffered reader for efficiency.
	reader := bufio.NewReader(stderr)
	var lineBuf []byte
	for {
		b, readErr := reader.ReadByte()
		if readErr != nil {
			break
		}
		if b == '\r' || b == '\n' {
			line := strings.TrimSpace(string(lineBuf))
			if line != "" && onProgress != nil {
				onProgress(line)
			}
			lineBuf = lineBuf[:0]
		} else {
			lineBuf = append(lineBuf, b)
		}
	}
	if line := strings.TrimSpace(string(lineBuf)); line != "" && onProgress != nil {
		onProgress(line)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	return nil
}

func RemoteAdd(dir, name, url string) error {
	return run(dir, "git", "remote", "add", name, url)
}

func RemoteSetURL(dir, name, url string) error {
	return run(dir, "git", "remote", "set-url", name, url)
}

func RemoteGetURL(dir, name string) (string, error) {
	return output(dir, "git", "remote", "get-url", name)
}

func ConfigSet(dir, key, value string) error {
	return run(dir, "git", "config", key, value)
}

func Fetch(dir, remote string) error {
	return run(dir, "git", "fetch", remote)
}

// PullRebase performs a pull with rebase from a remote branch.
func PullRebase(dir, remote, branch string) error {
	return run(dir, "git", "pull", "--rebase", remote, branch)
}

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

func BranchExists(dir, branch string) bool {
	_, err := output(dir, "git", "rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}

func CheckoutBranch(dir, branch string) error {
	return run(dir, "git", "checkout", branch)
}

// CreateBranchFrom creates a new branch from a start point and checks it out.
func CreateBranchFrom(dir, branch, startPoint string) error {
	return run(dir, "git", "checkout", "-b", branch, startPoint)
}

// WorktreeAdd adds a worktree at wtPath checking out an existing branch.
func WorktreeAdd(dir, wtPath, branch string) error {
	return run(dir, "git", "worktree", "add", wtPath, branch)
}

// WorktreeAddNew adds a worktree at wtPath and creates a new branch from startPoint.
func WorktreeAddNew(dir, wtPath, branch, startPoint string) error {
	return run(dir, "git", "worktree", "add", "-b", branch, wtPath, startPoint)
}

func DeleteBranch(dir, branch string) error {
	return run(dir, "git", "branch", "-D", branch)
}

// ListMergedBranches returns local branches fully merged into mergeBase (e.g. "origin/main").
// Branches currently checked out in a worktree (prefixed with "+ ") are excluded because
// git refuses to delete them while they are in use.
// Protected branches and the mergeBase branch name itself are NOT filtered here; callers must filter.
func ListMergedBranches(dir, mergeBase string) ([]string, error) {
	out, err := output(dir, "git", "branch", "--merged", mergeBase)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var branches []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		// Worktree-checked-out branches are prefixed with "+ "; skip them because
		// git refuses to delete a branch that is active in another worktree.
		if strings.HasPrefix(line, "+ ") {
			continue
		}
		// Strip the current-branch marker ("* ") to get the bare branch name.
		line = strings.TrimPrefix(line, "* ")
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

func DeleteRemoteBranch(dir, remote, branch string) error {
	return run(dir, "git", "push", remote, "--delete", branch)
}

func RemoteRefExists(dir, ref string) bool {
	_, err := output(dir, "git", "rev-parse", "--verify", "refs/remotes/"+ref)
	return err == nil
}

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
