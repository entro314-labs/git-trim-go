// Package git provides Git repository operations combining go-git with CLI fallback.
package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/dominikospritis/git-trim/internal/branch"
)

// Repository wraps a git repository with enhanced operations
type Repository struct {
	repo   *git.Repository
	path   string
	gitDir string
}

// Open opens a git repository at the given path
func Open(path string) (*Repository, error) {
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	gitDir := filepath.Join(worktree.Filesystem.Root(), ".git")

	return &Repository{
		repo:   repo,
		path:   worktree.Filesystem.Root(),
		gitDir: gitDir,
	}, nil
}

// OpenFromCwd opens a git repository from the current working directory
func OpenFromCwd() (*Repository, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	return Open(cwd)
}

// Path returns the repository root path
func (r *Repository) Path() string {
	return r.path
}

// GitDir returns the .git directory path
func (r *Repository) GitDir() string {
	return r.gitDir
}

// Head returns the current HEAD reference
func (r *Repository) Head() (*plumbing.Reference, error) {
	return r.repo.Head()
}

// CurrentBranch returns the current branch name, or empty string if detached
func (r *Repository) CurrentBranch() (string, error) {
	head, err := r.repo.Head()
	if err != nil {
		return "", err
	}
	if !head.Name().IsBranch() {
		return "", nil
	}
	return head.Name().Short(), nil
}

// IsDetached returns true if HEAD is detached
func (r *Repository) IsDetached() (bool, error) {
	head, err := r.repo.Head()
	if err != nil {
		return false, err
	}
	return !head.Name().IsBranch(), nil
}

// LocalBranches returns all local branches
func (r *Repository) LocalBranches() ([]branch.LocalBranch, error) {
	refs, err := r.repo.References()
	if err != nil {
		return nil, err
	}

	var branches []branch.LocalBranch
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branches = append(branches, branch.LocalBranch{
				Refname: ref.Name().String(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return branches, nil
}

// RemoteTrackingBranches returns all remote-tracking branches
func (r *Repository) RemoteTrackingBranches() ([]branch.RemoteTrackingBranch, error) {
	refs, err := r.repo.References()
	if err != nil {
		return nil, err
	}

	var branches []branch.RemoteTrackingBranch
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsRemote() {
			if strings.HasSuffix(ref.Name().String(), "/HEAD") {
				return nil
			}
			branches = append(branches, branch.RemoteTrackingBranch{
				Refname: ref.Name().String(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return branches, nil
}

// Remotes returns all configured remotes
func (r *Repository) Remotes() ([]string, error) {
	remotes, err := r.repo.Remotes()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(remotes))
	for i, remote := range remotes {
		names[i] = remote.Config().Name
	}
	return names, nil
}

// RemoteURL returns the URL for a remote
func (r *Repository) RemoteURL(name string) (string, error) {
	remote, err := r.repo.Remote(name)
	if err != nil {
		return "", err
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return "", fmt.Errorf("remote %s has no URLs", name)
	}
	return urls[0], nil
}

// BranchConfig returns the configuration for a branch
func (r *Repository) BranchConfig(branchName string) (*config.Branch, error) {
	cfg, err := r.repo.Config()
	if err != nil {
		return nil, err
	}

	bc, ok := cfg.Branches[branchName]
	if !ok {
		return nil, nil
	}
	return bc, nil
}

// TrackingBranches returns local branches with their upstream tracking info
func (r *Repository) TrackingBranches() ([]branch.TrackingBranch, error) {
	localBranches, err := r.LocalBranches()
	if err != nil {
		return nil, err
	}

	cfg, err := r.repo.Config()
	if err != nil {
		return nil, err
	}

	var trackingBranches []branch.TrackingBranch

	for _, lb := range localBranches {
		branchName := lb.Name()
		bc, ok := cfg.Branches[branchName]
		if !ok || bc.Remote == "" || bc.Merge.String() == "" {
			continue
		}

		remote, err := r.repo.Remote(bc.Remote)
		if err != nil {
			continue
		}

		remoteName := remote.Config().Name
		mergeBranch := strings.TrimPrefix(bc.Merge.String(), "refs/heads/")
		upstreamRef := fmt.Sprintf("refs/remotes/%s/%s", remoteName, mergeBranch)

		trackingBranches = append(trackingBranches, branch.TrackingBranch{
			Local:    lb,
			Upstream: branch.RemoteTrackingBranch{Refname: upstreamRef},
		})
	}

	return trackingBranches, nil
}

// DirectFetchBranches returns branches configured with hub-cli style direct fetch
func (r *Repository) DirectFetchBranches() ([]branch.DirectFetchBranch, error) {
	localBranches, err := r.LocalBranches()
	if err != nil {
		return nil, err
	}

	cfg, err := r.repo.Config()
	if err != nil {
		return nil, err
	}

	var directFetchBranches []branch.DirectFetchBranch

	for _, lb := range localBranches {
		branchName := lb.Name()
		bc, ok := cfg.Branches[branchName]
		if !ok || bc.Remote == "" {
			continue
		}

		_, err := r.repo.Remote(bc.Remote)
		if err == nil {
			continue
		}

		if bc.Merge.String() != "" {
			directFetchBranches = append(directFetchBranches, branch.DirectFetchBranch{
				Local: lb,
				Remote: branch.RemoteBranch{
					Remote:  bc.Remote,
					Refname: bc.Merge.String(),
				},
			})
		}
	}

	return directFetchBranches, nil
}

// NonTrackingLocalBranches returns local branches without upstream config
func (r *Repository) NonTrackingLocalBranches() ([]branch.LocalBranch, error) {
	localBranches, err := r.LocalBranches()
	if err != nil {
		return nil, err
	}

	cfg, err := r.repo.Config()
	if err != nil {
		return nil, err
	}

	var nonTracking []branch.LocalBranch

	for _, lb := range localBranches {
		branchName := lb.Name()
		bc, ok := cfg.Branches[branchName]
		if !ok || bc.Remote == "" {
			nonTracking = append(nonTracking, lb)
		}
	}

	return nonTracking, nil
}

// NonUpstreamRemoteTrackingBranches returns remote-tracking branches that aren't upstream of any local branch
func (r *Repository) NonUpstreamRemoteTrackingBranches() ([]branch.RemoteTrackingBranch, error) {
	remoteTB, err := r.RemoteTrackingBranches()
	if err != nil {
		return nil, err
	}

	trackingBranches, err := r.TrackingBranches()
	if err != nil {
		return nil, err
	}

	upstreamRefs := make(map[string]bool)
	for _, tb := range trackingBranches {
		upstreamRefs[tb.Upstream.Refname] = true
	}

	var orphans []branch.RemoteTrackingBranch
	for _, rtb := range remoteTB {
		if !upstreamRefs[rtb.Refname] {
			orphans = append(orphans, rtb)
		}
	}

	return orphans, nil
}

// ResolveRef resolves a reference to its commit hash
func (r *Repository) ResolveRef(refname string) (plumbing.Hash, error) {
	ref, err := r.repo.Reference(plumbing.ReferenceName(refname), true)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return ref.Hash(), nil
}

// RefExists checks if a reference exists
func (r *Repository) RefExists(refname string) bool {
	_, err := r.repo.Reference(plumbing.ReferenceName(refname), true)
	return err == nil
}

// MergeBase returns the merge base between two commits
func (r *Repository) MergeBase(commit1, commit2 plumbing.Hash) (plumbing.Hash, error) {
	c1, err := r.repo.CommitObject(commit1)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get commit %s: %w", commit1, err)
	}

	c2, err := r.repo.CommitObject(commit2)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get commit %s: %w", commit2, err)
	}

	bases, err := c1.MergeBase(c2)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to compute merge base: %w", err)
	}

	if len(bases) == 0 {
		return plumbing.ZeroHash, fmt.Errorf("no merge base found")
	}

	return bases[0].Hash, nil
}

// GetRemoteDefaultBranch returns the default branch for a remote
func (r *Repository) GetRemoteDefaultBranch(remoteName string) (string, error) {
	headRef := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/HEAD", remoteName))
	ref, err := r.repo.Reference(headRef, false)
	if err != nil {
		return "", err
	}

	if ref.Type() == plumbing.SymbolicReference {
		target := ref.Target().String()
		prefix := fmt.Sprintf("refs/remotes/%s/", remoteName)
		if strings.HasPrefix(target, prefix) {
			return strings.TrimPrefix(target, prefix), nil
		}
	}

	return "", fmt.Errorf("could not determine default branch for remote %s", remoteName)
}

// GoGitRepo returns the underlying go-git repository
func (r *Repository) GoGitRepo() *git.Repository {
	return r.repo
}
