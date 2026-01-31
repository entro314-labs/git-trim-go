package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dominikospritis/git-trim/internal/branch"
	"github.com/dominikospritis/git-trim/internal/git"
)

// Executor executes the trim plan
type Executor struct {
	repo        *git.Repository
	logger      *slog.Logger
	dryRun      bool
	allowDetach bool
}

// NewExecutor creates a new executor
func NewExecutor(repo *git.Repository, logger *slog.Logger, dryRun bool, allowDetach bool) *Executor {
	return &Executor{
		repo:        repo,
		logger:      logger,
		dryRun:      dryRun,
		allowDetach: allowDetach,
	}
}

// DeleteResult represents the result of a deletion operation
type DeleteResult struct {
	Branch  string
	Success bool
	Error   error
}

// Execute executes the trim plan
func (e *Executor) Execute(ctx context.Context, plan *TrimPlan) ([]DeleteResult, error) {
	var (
		results []DeleteResult
		errs    []error
	)

	currentBranch, err := e.repo.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	needsDetach := false
	for _, lb := range plan.LocalBranchesToDelete() {
		if lb.Name() == currentBranch {
			needsDetach = true
			break
		}
	}

	if needsDetach && !e.allowDetach {
		return nil, fmt.Errorf("refusing to delete current branch without detach permission")
	}

	if needsDetach && !e.dryRun {
		e.logger.Info("detaching HEAD before deleting current branch")
		if err := e.repo.DetachHead(ctx); err != nil {
			return nil, fmt.Errorf("failed to detach HEAD: %w", err)
		}
	}

	for _, lb := range plan.LocalBranchesToDelete() {
		result := e.deleteLocalBranch(ctx, lb)
		results = append(results, result)
		if result.Error != nil {
			errs = append(errs, result.Error)
		}
	}

	for _, rb := range plan.RemoteBranchesToDelete() {
		result := e.deleteRemoteBranch(ctx, rb)
		results = append(results, result)
		if result.Error != nil {
			errs = append(errs, result.Error)
		}
	}

	if len(errs) > 0 {
		return results, errors.Join(errs...)
	}

	return results, nil
}

func (e *Executor) deleteLocalBranch(ctx context.Context, lb branch.LocalBranch) DeleteResult {
	branchName := lb.Name()

	if e.dryRun {
		e.logger.Info("would delete local branch", "branch", branchName)
		return DeleteResult{Branch: branchName, Success: true}
	}

	e.logger.Info("deleting local branch", "branch", branchName)
	err := e.repo.DeleteLocalBranch(ctx, branchName, true)
	if err != nil {
		e.logger.Error("failed to delete local branch", "branch", branchName, "error", err)
		return DeleteResult{Branch: branchName, Success: false, Error: err}
	}

	return DeleteResult{Branch: branchName, Success: true}
}

func (e *Executor) deleteRemoteBranch(ctx context.Context, rb branch.RemoteBranch) DeleteResult {
	branchName := rb.Name()
	remote := rb.Remote

	if e.dryRun {
		e.logger.Info("would delete remote branch", "remote", remote, "branch", branchName)
		return DeleteResult{Branch: remote + "/" + branchName, Success: true}
	}

	e.logger.Info("deleting remote branch", "remote", remote, "branch", branchName)
	err := e.repo.DeleteRemoteBranch(ctx, remote, branchName)
	if err != nil {
		e.logger.Error("failed to delete remote branch", "remote", remote, "branch", branchName, "error", err)
		return DeleteResult{Branch: remote + "/" + branchName, Success: false, Error: err}
	}

	return DeleteResult{Branch: remote + "/" + branchName, Success: true}
}

// DryRun returns whether this is a dry run
func (e *Executor) DryRun() bool {
	return e.dryRun
}
