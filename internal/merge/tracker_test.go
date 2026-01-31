package merge_test

import (
	"context"
	"testing"

	"github.com/dominikospritis/git-trim/internal/git"
	"github.com/dominikospritis/git-trim/internal/merge"
	"github.com/dominikospritis/git-trim/internal/testutil"
)

func TestMergeTracker_NoFFMerge(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	// Create a feature branch with a commit
	repo.CreateBranch("feature")
	repo.AddCommit("feature.txt", "feature content", "Add feature")
	repo.Push()

	// Switch to main and merge with --no-ff
	repo.Checkout("main")
	repo.Git("pull", "origin", "main")
	repo.MergeNoFF("feature")
	repo.Push()

	// Delete remote feature branch
	repo.DeleteRemoteBranch("feature")
	repo.Fetch()

	// Open the repository with our git package
	gitRepo, err := git.Open(repo.WorkDir)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	tracker := merge.NewTracker(gitRepo)
	ctx := context.Background()

	// The feature branch should be detected as merged
	style, err := tracker.IsMergedInto(ctx, "refs/heads/feature", "refs/heads/main")
	if err != nil {
		t.Fatalf("IsMergedInto returned error: %v", err)
	}
	if style != merge.NoFFMerged {
		t.Errorf("expected NoFFMerged, got %v", style)
	}
}

func TestMergeTracker_SquashMerge(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	// Create a feature branch with commits
	repo.CreateBranch("feature")
	repo.AddCommit("feature1.txt", "content 1", "First feature commit")
	repo.AddCommit("feature2.txt", "content 2", "Second feature commit")
	repo.Push()

	// Switch to main and squash merge
	repo.Checkout("main")
	repo.Git("pull", "origin", "main")
	repo.MergeSquash("feature")
	repo.Push()

	// Delete remote feature branch
	repo.DeleteRemoteBranch("feature")
	repo.Fetch()

	// Open the repository with our git package
	gitRepo, err := git.Open(repo.WorkDir)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	tracker := merge.NewTracker(gitRepo)
	ctx := context.Background()

	// The feature branch should be detected as squash merged
	style, err := tracker.IsMergedInto(ctx, "refs/heads/feature", "refs/heads/main")
	if err != nil {
		t.Fatalf("IsMergedInto returned error: %v", err)
	}
	if style != merge.SquashMerged {
		t.Errorf("expected SquashMerged, got %v", style)
	}
}

func TestMergeTracker_RebaseMerge(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	// Create a feature branch with a commit
	repo.CreateBranch("feature")
	repo.AddCommit("feature.txt", "feature content", "Add feature")

	// Get the commit hash of the feature branch tip
	featureRef := repo.Git("rev-parse", "HEAD")

	// Push feature to remote
	repo.Push()

	// Switch to main and add another commit to create divergence
	repo.Checkout("main")
	repo.AddCommit("main.txt", "main content", "Main commit")
	repo.Push()

	// Rebase feature onto main
	repo.Checkout("feature")
	repo.Rebase("main")
	repo.PushForce()

	// Switch to main and fast-forward merge the rebased feature
	repo.Checkout("main")
	repo.Git("merge", "--ff-only", "feature")
	repo.Push()

	// Delete remote feature branch
	repo.DeleteRemoteBranch("feature")
	repo.Fetch()

	// Open the repository with our git package
	gitRepo, err := git.Open(repo.WorkDir)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	tracker := merge.NewTracker(gitRepo)
	ctx := context.Background()

	// The feature branch should be detected as rebase merged
	// Note: The local feature branch still points to the rebased commit
	style, err := tracker.IsMergedInto(ctx, "refs/heads/feature", "refs/heads/main")
	if err != nil {
		t.Fatalf("IsMergedInto returned error: %v", err)
	}

	// After rebase and ff merge, it should be either NoFF (if commits match) or Rebase
	if style == merge.NotMerged {
		t.Errorf("expected branch to be detected as merged, got NotMerged")
	}
	t.Logf("Merge style: %v (original feature ref: %s)", style, featureRef)
}

func TestMergeTracker_NotMerged(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	// Create a feature branch with commits
	repo.CreateBranch("feature")
	repo.AddCommit("feature.txt", "feature content", "Add feature")

	// Don't merge it into main

	// Open the repository with our git package
	gitRepo, err := git.Open(repo.WorkDir)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	tracker := merge.NewTracker(gitRepo)
	ctx := context.Background()

	// The feature branch should NOT be detected as merged
	style, err := tracker.IsMergedInto(ctx, "refs/heads/feature", "refs/heads/main")
	if err != nil {
		t.Fatalf("IsMergedInto returned error: %v", err)
	}
	if style != merge.NotMerged {
		t.Errorf("expected NotMerged, got %v", style)
	}
}

func TestMergeTracker_IsMergedMultipleBases(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	// Create develop branch
	repo.CreateBranch("develop")
	repo.AddCommit("develop.txt", "develop content", "Init develop")
	repo.Push()

	// Create feature branch from develop
	repo.CreateBranch("feature")
	repo.AddCommit("feature.txt", "feature content", "Add feature")
	repo.Push()

	// Merge feature into develop
	repo.Checkout("develop")
	repo.MergeNoFF("feature")
	repo.Push()

	// Delete remote feature branch
	repo.DeleteRemoteBranch("feature")
	repo.Fetch()

	// Open the repository
	gitRepo, err := git.Open(repo.WorkDir)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	tracker := merge.NewTracker(gitRepo)
	ctx := context.Background()

	// Check against multiple bases
	bases := []string{"refs/heads/main", "refs/heads/develop"}
	style, base, err := tracker.IsMerged(ctx, "refs/heads/feature", bases)
	if err != nil {
		t.Fatalf("IsMerged returned error: %v", err)
	}

	if style == merge.NotMerged {
		t.Error("expected branch to be detected as merged into develop")
	}

	if base != "refs/heads/develop" {
		t.Errorf("expected base to be develop, got %s", base)
	}
}
