package git

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RunGit executes a git command and returns the output
func (r *Repository) RunGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RunGitWithStdin executes a git command with stdin input
func (r *Repository) RunGitWithStdin(ctx context.Context, stdin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.path
	cmd.Stdin = strings.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RunGitLines executes a git command and returns output as lines
func (r *Repository) RunGitLines(ctx context.Context, args ...string) ([]string, error) {
	output, err := r.RunGit(ctx, args...)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// UpdateRemotes runs git remote update --prune
func (r *Repository) UpdateRemotes(ctx context.Context) error {
	_, err := r.RunGit(ctx, "remote", "update", "--prune")
	return err
}

// UpdateRemote runs git fetch --prune for a specific remote
func (r *Repository) UpdateRemote(ctx context.Context, remote string) error {
	_, err := r.RunGit(ctx, "fetch", "--prune", remote)
	return err
}

// GetFetchHeadTime returns the modification time of FETCH_HEAD
func (r *Repository) GetFetchHeadTime() (time.Time, error) {
	fetchHeadPath := filepath.Join(r.gitDir, "FETCH_HEAD")
	info, err := os.Stat(fetchHeadPath)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// ShouldUpdate returns true if enough time has passed since last update
func (r *Repository) ShouldUpdate(interval time.Duration) (bool, error) {
	lastUpdate, err := r.GetFetchHeadTime()
	if err != nil {
		return true, err
	}
	if lastUpdate.IsZero() {
		return true, nil
	}
	return time.Since(lastUpdate) > interval, nil
}

// DeleteLocalBranch deletes a local branch
func (r *Repository) DeleteLocalBranch(ctx context.Context, branchName string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := r.RunGit(ctx, "branch", flag, branchName)
	return err
}

// DeleteRemoteBranch deletes a branch on a remote
func (r *Repository) DeleteRemoteBranch(ctx context.Context, remote, branchName string) error {
	_, err := r.RunGit(ctx, "push", "--delete", remote, branchName)
	return err
}

// Checkout checks out a ref
func (r *Repository) Checkout(ctx context.Context, ref string) error {
	_, err := r.RunGit(ctx, "checkout", ref)
	return err
}

// DetachHead detaches HEAD at the current commit
func (r *Repository) DetachHead(ctx context.Context) error {
	_, err := r.RunGit(ctx, "checkout", "--detach")
	return err
}

// GetMergedBranches returns branches merged into the given base
func (r *Repository) GetMergedBranches(ctx context.Context, base string) ([]string, error) {
	output, err := r.RunGit(ctx, "branch", "--merged", base, "--format=%(refname)")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return nil, nil
	}

	var branches []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		ref := strings.TrimSpace(scanner.Text())
		if ref != "" {
			branches = append(branches, ref)
		}
	}
	return branches, scanner.Err()
}

// IsRebaseMerged checks if a branch is rebase-merged into base
func (r *Repository) IsRebaseMerged(ctx context.Context, base, branch string) (bool, error) {
	output, err := r.RunGit(ctx, "rev-list", "--cherry-pick", "--right-only", "--no-merges", "-n1", fmt.Sprintf("%s...%s", base, branch))
	if err != nil {
		return false, err
	}
	return output == "", nil
}

// IsSquashMerged checks if a branch was squash-merged into base
func (r *Repository) IsSquashMerged(ctx context.Context, base, branch string) (bool, error) {
	mergeBase, err := r.RunGit(ctx, "merge-base", base, branch)
	if err != nil {
		return false, err
	}

	tree, err := r.RunGit(ctx, "rev-parse", branch+"^{tree}")
	if err != nil {
		return false, err
	}

	tempCommit, err := r.RunGitWithStdin(ctx, "Temporary commit for squash detection", "commit-tree", tree, "-p", mergeBase)
	if err != nil {
		return false, err
	}

	output, err := r.RunGit(ctx, "rev-list", "--cherry-pick", "--right-only", "--no-merges", "-n1", fmt.Sprintf("%s...%s", base, tempCommit))
	if err != nil {
		return false, err
	}

	return output == "", nil
}

// LsRemoteHeads returns the heads from a remote using ls-remote
func (r *Repository) LsRemoteHeads(ctx context.Context, remote string) (map[string]string, error) {
	output, err := r.RunGit(ctx, "ls-remote", "--heads", remote)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			hash := parts[0]
			ref := parts[1]
			result[ref] = hash
		}
	}
	return result, scanner.Err()
}

// WorktreeInfo holds information about a git worktree
type WorktreeInfo struct {
	Path   string
	Branch string
}

// Worktrees returns the list of worktrees
func (r *Repository) Worktrees(ctx context.Context) ([]WorktreeInfo, error) {
	output, err := r.RunGit(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []WorktreeInfo
	var current WorktreeInfo

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch ")
		}
	}

	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, scanner.Err()
}

// WorktreeBranches returns a set of branch refs attached to worktrees
func (r *Repository) WorktreeBranches(ctx context.Context) (map[string]string, error) {
	worktrees, err := r.Worktrees(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, wt := range worktrees {
		if wt.Branch != "" {
			result[wt.Branch] = wt.Path
		}
	}
	return result, nil
}

// RevParse runs git rev-parse on a revision
func (r *Repository) RevParse(ctx context.Context, rev string) (string, error) {
	return r.RunGit(ctx, "rev-parse", rev)
}

// IsAncestor checks if ancestor is an ancestor of descendant
func (r *Repository) IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error) {
	err := exec.CommandContext(ctx, "git", "-C", r.path, "merge-base", "--is-ancestor", ancestor, descendant).Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}
