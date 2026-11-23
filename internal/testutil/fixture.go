// Package testutil provides utilities for integration testing.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRepo represents a test git repository
type TestRepo struct {
	T       *testing.T
	RootDir string
	WorkDir string
	Origin  string
}

// NewTestRepo creates a new test repository with an origin
func NewTestRepo(t *testing.T) *TestRepo {
	t.Helper()

	rootDir := t.TempDir()
	originDir := filepath.Join(rootDir, "origin.git")
	workDir := filepath.Join(rootDir, "work")

	if err := os.MkdirAll(originDir, 0755); err != nil {
		t.Fatalf("failed to create origin dir: %v", err)
	}

	cmd := exec.Command("git", "init", "--bare", "--initial-branch=main")
	cmd.Dir = originDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to init origin: %v\n%s", err, out)
	}

	cmd = exec.Command("git", "clone", originDir, workDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to clone: %v\n%s", err, out)
	}

	repo := &TestRepo{
		T:       t,
		RootDir: rootDir,
		WorkDir: workDir,
		Origin:  originDir,
	}

	// The tests refer to the default branch as "main"; do not depend on the
	// host's init.defaultBranch setting.
	repo.Git("symbolic-ref", "HEAD", "refs/heads/main")
	repo.Git("config", "user.email", "test@example.com")
	repo.Git("config", "user.name", "Test User")

	repo.WriteFile("README.md", "# Test Repository\n")
	repo.Git("add", ".")
	repo.Git("commit", "-m", "Initial commit")
	repo.Git("push", "-u", "origin", "HEAD")

	return repo
}

// Git runs a git command in the work directory
func (r *TestRepo) Git(args ...string) string {
	r.T.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.WorkDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.T.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

// GitInOrigin runs a git command in the origin directory
func (r *TestRepo) GitInOrigin(args ...string) string {
	r.T.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Origin
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.T.Fatalf("git %v in origin failed: %v\n%s", args, err, out)
	}
	return string(out)
}

// WriteFile writes a file in the work directory
func (r *TestRepo) WriteFile(name, content string) {
	r.T.Helper()
	path := filepath.Join(r.WorkDir, name)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		r.T.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		r.T.Fatalf("failed to write file: %v", err)
	}
}

// CreateBranch creates and checks out a new branch
func (r *TestRepo) CreateBranch(name string) {
	r.T.Helper()
	r.Git("checkout", "-b", name)
}

// Checkout checks out a branch
func (r *TestRepo) Checkout(name string) {
	r.T.Helper()
	r.Git("checkout", name)
}

// AddCommit adds a file and commits
func (r *TestRepo) AddCommit(filename, content, message string) {
	r.T.Helper()
	r.WriteFile(filename, content)
	r.Git("add", filename)
	r.Git("commit", "-m", message)
}

// Push pushes the current branch to origin
func (r *TestRepo) Push() {
	r.T.Helper()
	r.Git("push", "-u", "origin", "HEAD")
}

// PushForce force pushes the current branch to origin
func (r *TestRepo) PushForce() {
	r.T.Helper()
	r.Git("push", "-f", "origin", "HEAD")
}

// Fetch fetches from origin
func (r *TestRepo) Fetch() {
	r.T.Helper()
	r.Git("fetch", "--prune", "origin")
}

// MergeNoFF merges a branch with --no-ff
func (r *TestRepo) MergeNoFF(branch string) {
	r.T.Helper()
	r.Git("merge", "--no-ff", "-m", "Merge "+branch, branch)
}

// MergeSquash squash merges a branch
func (r *TestRepo) MergeSquash(branch string) {
	r.T.Helper()
	r.Git("merge", "--squash", branch)
	r.Git("commit", "-m", "Squash merge "+branch)
}

// Rebase rebases current branch onto another
func (r *TestRepo) Rebase(onto string) {
	r.T.Helper()
	r.Git("rebase", onto)
}

// DeleteRemoteBranch deletes a branch on origin
func (r *TestRepo) DeleteRemoteBranch(name string) {
	r.T.Helper()
	r.Git("push", "origin", "--delete", name)
}

// CurrentBranch returns the current branch name
func (r *TestRepo) CurrentBranch() string {
	r.T.Helper()
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = r.WorkDir
	out, err := cmd.Output()
	if err != nil {
		r.T.Fatalf("failed to get current branch: %v", err)
	}
	return string(out[:len(out)-1])
}

// BranchExists checks if a local branch exists
func (r *TestRepo) BranchExists(name string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	cmd.Dir = r.WorkDir
	return cmd.Run() == nil
}

// RemoteBranchExists checks if a remote tracking branch exists
func (r *TestRepo) RemoteBranchExists(name string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+name)
	cmd.Dir = r.WorkDir
	return cmd.Run() == nil
}
