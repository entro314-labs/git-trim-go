// Package merge provides merge detection algorithms.
package merge

import (
	"context"
	"sync"

	"github.com/dominikospritis/git-trim/internal/git"
)

// MergeStyle represents how a branch was merged
type MergeStyle int

const (
	NotMerged MergeStyle = iota
	NoFFMerged
	RebaseMerged
	SquashMerged
)

// String returns a human-readable representation of the merge style
func (m MergeStyle) String() string {
	switch m {
	case NotMerged:
		return "not merged"
	case NoFFMerged:
		return "merged (no-ff)"
	case RebaseMerged:
		return "merged (rebase/ff)"
	case SquashMerged:
		return "merged (squash)"
	default:
		return "unknown"
	}
}

// IsMerged returns true if the style indicates the branch is merged
func (m MergeStyle) IsMerged() bool {
	return m != NotMerged
}

// Tracker tracks merge status of branches against base branches
type Tracker struct {
	repo   *git.Repository
	mu     sync.RWMutex
	caches map[string]*baseCache
}

type baseCache struct {
	noffMerged   map[string]bool
	rebaseMerged map[string]bool
	squashMerged map[string]bool
}

func newBaseCache() *baseCache {
	return &baseCache{
		noffMerged:   make(map[string]bool),
		rebaseMerged: make(map[string]bool),
		squashMerged: make(map[string]bool),
	}
}

// NewTracker creates a new merge tracker
func NewTracker(repo *git.Repository) *Tracker {
	return &Tracker{
		repo:   repo,
		caches: make(map[string]*baseCache),
	}
}

func (t *Tracker) getOrCreateCache(base string) *baseCache {
	t.mu.Lock()
	defer t.mu.Unlock()

	if cache, ok := t.caches[base]; ok {
		return cache
	}

	cache := newBaseCache()
	t.caches[base] = cache
	return cache
}

// IsMerged checks if a branch is merged into any of the base branches
func (t *Tracker) IsMerged(ctx context.Context, branchRef string, bases []string) (MergeStyle, string, error) {
	for _, base := range bases {
		style, err := t.IsMergedInto(ctx, branchRef, base)
		if err != nil {
			return NotMerged, "", err
		}
		if style.IsMerged() {
			return style, base, nil
		}
	}
	return NotMerged, "", nil
}

// IsMergedInto checks if a branch is merged into a specific base branch
func (t *Tracker) IsMergedInto(ctx context.Context, branchRef, base string) (MergeStyle, error) {
	cache := t.getOrCreateCache(base)

	t.mu.RLock()
	if merged, ok := cache.noffMerged[branchRef]; ok && merged {
		t.mu.RUnlock()
		return NoFFMerged, nil
	}
	if merged, ok := cache.rebaseMerged[branchRef]; ok && merged {
		t.mu.RUnlock()
		return RebaseMerged, nil
	}
	if merged, ok := cache.squashMerged[branchRef]; ok && merged {
		t.mu.RUnlock()
		return SquashMerged, nil
	}
	t.mu.RUnlock()

	if merged, err := t.isNoFFMerged(ctx, branchRef, base); err != nil {
		return NotMerged, err
	} else if merged {
		t.mu.Lock()
		cache.noffMerged[branchRef] = true
		t.mu.Unlock()
		return NoFFMerged, nil
	}

	if merged, err := t.isRebaseMerged(ctx, branchRef, base); err != nil {
		return NotMerged, err
	} else if merged {
		t.mu.Lock()
		cache.rebaseMerged[branchRef] = true
		t.mu.Unlock()
		return RebaseMerged, nil
	}

	if merged, err := t.isSquashMerged(ctx, branchRef, base); err != nil {
		return NotMerged, err
	} else if merged {
		t.mu.Lock()
		cache.squashMerged[branchRef] = true
		t.mu.Unlock()
		return SquashMerged, nil
	}

	return NotMerged, nil
}

func (t *Tracker) isNoFFMerged(ctx context.Context, branchRef, base string) (bool, error) {
	mergedBranches, err := t.repo.GetMergedBranches(ctx, base)
	if err != nil {
		return false, err
	}

	for _, merged := range mergedBranches {
		if merged == branchRef {
			return true, nil
		}
	}
	return false, nil
}

func (t *Tracker) isRebaseMerged(ctx context.Context, branchRef, base string) (bool, error) {
	merged, err := t.repo.IsRebaseMerged(ctx, base, branchRef)
	if err != nil {
		return false, err
	}
	return merged, nil
}

func (t *Tracker) isSquashMerged(ctx context.Context, branchRef, base string) (bool, error) {
	merged, err := t.repo.IsSquashMerged(ctx, base, branchRef)
	if err != nil {
		return false, err
	}
	return merged, nil
}

// PreloadNoFFMerged preloads the no-ff merge cache for a base branch
func (t *Tracker) PreloadNoFFMerged(ctx context.Context, base string) error {
	mergedBranches, err := t.repo.GetMergedBranches(ctx, base)
	if err != nil {
		return err
	}

	cache := t.getOrCreateCache(base)

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, branch := range mergedBranches {
		cache.noffMerged[branch] = true
	}

	return nil
}
