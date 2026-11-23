// Package core provides the core branch trimming logic.
package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dominikospritis/git-trim/internal/branch"
	"github.com/dominikospritis/git-trim/internal/config"
	"github.com/dominikospritis/git-trim/internal/git"
	"github.com/dominikospritis/git-trim/internal/merge"
)

// Classifier classifies branches for potential deletion
type Classifier struct {
	repo    *git.Repository
	tracker *merge.Tracker
	cfg     *config.Config
	bases   []string
	logger  *slog.Logger
}

// NewClassifier creates a new branch classifier
func NewClassifier(repo *git.Repository, cfg *config.Config, bases []string, logger *slog.Logger) *Classifier {
	return &Classifier{
		repo:    repo,
		tracker: merge.NewTracker(repo),
		cfg:     cfg,
		bases:   bases,
		logger:  logger,
	}
}

// ClassifyAll classifies all branches in the repository
func (c *Classifier) ClassifyAll(ctx context.Context) ([]branch.ClassifiedBranch, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var results []branch.ClassifiedBranch
	var mu sync.Mutex
	var wg sync.WaitGroup
	errCh := make(chan error, 4)

	for _, base := range c.bases {
		if err := c.tracker.PreloadNoFFMerged(ctx, base); err != nil {
			cancel()
			return nil, fmt.Errorf("preload merge cache for %s: %w", base, err)
		}
	}

	collect := func(fn func(context.Context) ([]branch.ClassifiedBranch, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			classified, err := fn(ctx)
			if err != nil {
				cancel()
				errCh <- err
				return
			}
			mu.Lock()
			results = append(results, classified...)
			mu.Unlock()
		}()
	}

	collect(func(ctx context.Context) ([]branch.ClassifiedBranch, error) {
		classified, err := c.classifyTrackingBranches(ctx)
		if err != nil {
			return nil, fmt.Errorf("tracking branches: %w", err)
		}
		return classified, nil
	})

	collect(func(ctx context.Context) ([]branch.ClassifiedBranch, error) {
		classified, err := c.classifyDirectFetchBranches(ctx)
		if err != nil {
			return nil, fmt.Errorf("direct fetch branches: %w", err)
		}
		return classified, nil
	})

	collect(func(ctx context.Context) ([]branch.ClassifiedBranch, error) {
		classified, err := c.classifyNonTrackingLocalBranches(ctx)
		if err != nil {
			return nil, fmt.Errorf("non-tracking branches: %w", err)
		}
		return classified, nil
	})

	collect(func(ctx context.Context) ([]branch.ClassifiedBranch, error) {
		classified, err := c.classifyNonUpstreamRemoteTrackingBranches(ctx)
		if err != nil {
			return nil, fmt.Errorf("orphan remote-tracking branches: %w", err)
		}
		return classified, nil
	})

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return results, nil
}

func (c *Classifier) classifyTrackingBranches(ctx context.Context) ([]branch.ClassifiedBranch, error) {
	trackingBranches, err := c.repo.TrackingBranches()
	if err != nil {
		return nil, err
	}

	var results []branch.ClassifiedBranch

	for _, tb := range trackingBranches {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		classified, err := c.classifyTrackingBranch(ctx, tb)
		if err != nil {
			return results, err
		}
		results = append(results, classified...)
	}

	return results, nil
}

func (c *Classifier) classifyTrackingBranch(ctx context.Context, tb branch.TrackingBranch) ([]branch.ClassifiedBranch, error) {
	var results []branch.ClassifiedBranch

	upstreamExists := c.repo.RefExists(tb.Upstream.Refname)
	localMerged, _, err := c.tracker.IsMerged(ctx, tb.Local.Refname, c.bases)
	if err != nil {
		return nil, err
	}

	if !upstreamExists {
		if localMerged.IsMerged() {
			results = append(results, branch.MergedLocal{Branch: tb.Local})
		} else {
			results = append(results, branch.Stray{Branch: tb.Local})
		}
		return results, nil
	}

	upstreamMerged, _, err := c.tracker.IsMerged(ctx, tb.Upstream.Refname, c.bases)
	if err != nil {
		return nil, err
	}

	if localMerged.IsMerged() && upstreamMerged.IsMerged() {
		results = append(results, branch.MergedLocal{Branch: tb.Local})
		results = append(results, branch.MergedRemoteTracking{Branch: tb.Upstream})
	} else if localMerged.IsMerged() && !upstreamMerged.IsMerged() {
		results = append(results, branch.DivergedRemoteTracking(tb))
	} else if !localMerged.IsMerged() && upstreamMerged.IsMerged() {
		results = append(results, branch.Stray{Branch: tb.Local})
		results = append(results, branch.MergedRemoteTracking{Branch: tb.Upstream})
	}

	return results, nil
}

func (c *Classifier) classifyDirectFetchBranches(ctx context.Context) ([]branch.ClassifiedBranch, error) {
	directFetchBranches, err := c.repo.DirectFetchBranches()
	if err != nil {
		return nil, err
	}

	var results []branch.ClassifiedBranch

	for _, dfb := range directFetchBranches {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		classified, err := c.classifyDirectFetchBranch(ctx, dfb)
		if err != nil {
			return results, err
		}
		results = append(results, classified...)
	}

	return results, nil
}

func (c *Classifier) classifyDirectFetchBranch(ctx context.Context, dfb branch.DirectFetchBranch) ([]branch.ClassifiedBranch, error) {
	var results []branch.ClassifiedBranch

	localMerged, _, err := c.tracker.IsMerged(ctx, dfb.Local.Refname, c.bases)
	if err != nil {
		return nil, err
	}

	remoteHeads, err := c.repo.LsRemoteHeads(ctx, dfb.Remote.Remote)
	if err != nil {
		return nil, fmt.Errorf("ls-remote %s: %w", dfb.Remote.Remote, err)
	}

	remoteRef := dfb.Remote.Refname
	remoteExists := false
	remoteHash := ""
	if hash, ok := remoteHeads[remoteRef]; ok {
		remoteExists = true
		remoteHash = hash
	}

	if !remoteExists {
		if localMerged.IsMerged() {
			results = append(results, branch.MergedDirectFetch(dfb))
		}
		return results, nil
	}

	localHash, err := c.repo.RevParse(ctx, dfb.Local.Refname)
	if err != nil {
		return nil, err
	}

	if localMerged.IsMerged() {
		if localHash == remoteHash {
			results = append(results, branch.MergedDirectFetch(dfb))
		} else {
			results = append(results, branch.DivergedDirectFetch(dfb))
		}
	}

	return results, nil
}

func (c *Classifier) classifyNonTrackingLocalBranches(ctx context.Context) ([]branch.ClassifiedBranch, error) {
	nonTrackingBranches, err := c.repo.NonTrackingLocalBranches()
	if err != nil {
		return nil, err
	}

	var results []branch.ClassifiedBranch

	for _, lb := range nonTrackingBranches {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		merged, _, err := c.tracker.IsMerged(ctx, lb.Refname, c.bases)
		if err != nil {
			return results, err
		}
		if merged.IsMerged() {
			results = append(results, branch.MergedNonTrackingLocal{Branch: lb})
		}
	}

	return results, nil
}

func (c *Classifier) classifyNonUpstreamRemoteTrackingBranches(ctx context.Context) ([]branch.ClassifiedBranch, error) {
	orphanBranches, err := c.repo.NonUpstreamRemoteTrackingBranches()
	if err != nil {
		return nil, err
	}

	var results []branch.ClassifiedBranch

	for _, rtb := range orphanBranches {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		merged, _, err := c.tracker.IsMerged(ctx, rtb.Refname, c.bases)
		if err != nil {
			return results, err
		}
		if merged.IsMerged() {
			results = append(results, branch.MergedNonUpstreamRemoteTracking{Branch: rtb})
		}
	}

	return results, nil
}
