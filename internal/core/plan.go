package core

import (
	"fmt"
	"strings"

	"github.com/dominikospritis/git-trim/internal/branch"
	"github.com/dominikospritis/git-trim/internal/config"
)

// PreserveReason explains why a branch was preserved
type PreserveReason int

const (
	PreserveBase PreserveReason = iota
	PreserveProtected
	PreserveWorktree
	PreserveHEAD
	PreserveNotInFilter
)

func (r PreserveReason) String() string {
	switch r {
	case PreserveBase:
		return "base branch"
	case PreserveProtected:
		return "protected by pattern"
	case PreserveWorktree:
		return "attached to worktree"
	case PreserveHEAD:
		return "current HEAD"
	case PreserveNotInFilter:
		return "not in delete filter"
	default:
		return "unknown"
	}
}

// Preserved represents a branch that was preserved with its reason
type Preserved struct {
	Branch  branch.ClassifiedBranch
	Reason  PreserveReason
	Details string
}

// SkipSuggestion explains why a branch was skipped
type SkipSuggestion struct {
	Reason     string
	Suggestion string
}

// TrimPlan holds the plan for which branches to delete
type TrimPlan struct {
	ToDelete  []branch.ClassifiedBranch
	Preserved []Preserved
	Skipped   map[string]SkipSuggestion
}

// NewTrimPlan creates a new empty trim plan
func NewTrimPlan() *TrimPlan {
	return &TrimPlan{
		ToDelete:  make([]branch.ClassifiedBranch, 0),
		Preserved: make([]Preserved, 0),
		Skipped:   make(map[string]SkipSuggestion),
	}
}

// IsEmpty returns true if there's nothing to delete
func (p *TrimPlan) IsEmpty() bool {
	return len(p.ToDelete) == 0
}

// LocalBranchesToDelete returns local branches marked for deletion
func (p *TrimPlan) LocalBranchesToDelete() []branch.LocalBranch {
	var result []branch.LocalBranch
	seen := make(map[string]bool)

	for _, cb := range p.ToDelete {
		if lb := cb.LocalBranch(); lb != nil {
			if !seen[lb.Refname] {
				seen[lb.Refname] = true
				result = append(result, *lb)
			}
		}
	}

	return result
}

// RemoteBranchesToDelete returns remote branches marked for deletion
func (p *TrimPlan) RemoteBranchesToDelete() []branch.RemoteBranch {
	var result []branch.RemoteBranch
	seen := make(map[string]bool)

	for _, cb := range p.ToDelete {
		if rb := cb.RemoteBranch(); rb != nil {
			key := rb.Remote + ":" + rb.Refname
			if !seen[key] {
				seen[key] = true
				result = append(result, *rb)
			}
		}
	}

	return result
}

// PlanBuilder builds a trim plan from classified branches
type PlanBuilder struct {
	cfg              *config.Config
	bases            []string
	worktreeBranches map[string]string
	currentBranch    string
	protectedMatcher *config.GlobMatcher
}

// NewPlanBuilder creates a new plan builder
func NewPlanBuilder(cfg *config.Config, bases []string, worktreeBranches map[string]string, currentBranch string) *PlanBuilder {
	return &PlanBuilder{
		cfg:              cfg,
		bases:            bases,
		worktreeBranches: worktreeBranches,
		currentBranch:    currentBranch,
		protectedMatcher: config.NewGlobMatcher(cfg.Protected),
	}
}

// Build creates a trim plan from classified branches
func (b *PlanBuilder) Build(classified []branch.ClassifiedBranch) *TrimPlan {
	plan := NewTrimPlan()

	baseSet := make(map[string]bool)
	for _, base := range b.bases {
		baseSet["refs/heads/"+base] = true
		baseSet[base] = true
	}

	for _, cb := range classified {
		if !b.matchesDeleteFilter(cb) {
			plan.Preserved = append(plan.Preserved, Preserved{
				Branch: cb,
				Reason: PreserveNotInFilter,
			})
			continue
		}

		if reason, details, preserved := b.shouldPreserve(cb, baseSet); preserved {
			plan.Preserved = append(plan.Preserved, Preserved{
				Branch:  cb,
				Reason:  reason,
				Details: details,
			})
			continue
		}

		plan.ToDelete = append(plan.ToDelete, cb)
	}

	return plan
}

func (b *PlanBuilder) matchesDeleteFilter(cb branch.ClassifiedBranch) bool {
	switch cb.Type() {
	case branch.TypeMergedLocal:
		return b.cfg.Delete.MergedLocal

	case branch.TypeStray:
		return b.cfg.Delete.Stray

	case branch.TypeMergedRemoteTracking:
		rtb := cb.RemoteTrackingBranch()
		if rtb == nil {
			return false
		}
		return b.cfg.Delete.MergedRemote.Matches(rtb.RemoteName())

	case branch.TypeDivergedRemoteTracking:
		drt, ok := cb.(branch.DivergedRemoteTracking)
		if !ok {
			return false
		}
		return b.cfg.Delete.Diverged.Matches(drt.Upstream.RemoteName())

	case branch.TypeMergedDirectFetch:
		return b.cfg.Delete.MergedLocal

	case branch.TypeDivergedDirectFetch:
		ddf, ok := cb.(branch.DivergedDirectFetch)
		if !ok {
			return false
		}
		return b.cfg.Delete.Diverged.Matches(ddf.Remote.Remote)

	case branch.TypeMergedNonTrackingLocal:
		return b.cfg.Delete.MergedNonTrackingLocal

	case branch.TypeMergedNonUpstreamRemoteTracking:
		rtb := cb.RemoteTrackingBranch()
		if rtb == nil {
			return false
		}
		return b.cfg.Delete.MergedNonUpstreamRemoteTracking.Matches(rtb.RemoteName())
	}

	return false
}

func (b *PlanBuilder) shouldPreserve(cb branch.ClassifiedBranch, baseSet map[string]bool) (PreserveReason, string, bool) {
	if lb := cb.LocalBranch(); lb != nil {
		if baseSet[lb.Refname] || baseSet[lb.Name()] {
			return PreserveBase, "", true
		}

		if lb.Name() == b.currentBranch || lb.Refname == "refs/heads/"+b.currentBranch {
			if !b.cfg.Detach {
				return PreserveHEAD, "", true
			}
		}

		if b.protectedMatcher.Matches(lb.Name()) {
			return PreserveProtected, "", true
		}

		if wtPath, ok := b.worktreeBranches[lb.Refname]; ok {
			return PreserveWorktree, wtPath, true
		}
	}

	if rtb := cb.RemoteTrackingBranch(); rtb != nil {
		if b.protectedMatcher.Matches(rtb.BranchName()) {
			return PreserveProtected, "", true
		}
	}

	return 0, "", false
}

// Summary returns a human-readable summary of the plan
func (p *TrimPlan) Summary() string {
	var sb strings.Builder

	if len(p.ToDelete) > 0 {
		sb.WriteString(fmt.Sprintf("Branches to delete: %d\n", len(p.ToDelete)))
		for _, cb := range p.ToDelete {
			sb.WriteString(fmt.Sprintf("  - %s\n", cb.Description()))
		}
	}

	if len(p.Preserved) > 0 {
		sb.WriteString(fmt.Sprintf("\nPreserved branches: %d\n", len(p.Preserved)))
		for _, pv := range p.Preserved {
			details := ""
			if pv.Details != "" {
				details = fmt.Sprintf(" (%s)", pv.Details)
			}
			sb.WriteString(fmt.Sprintf("  - %s: %s%s\n", pv.Branch.Description(), pv.Reason, details))
		}
	}

	return sb.String()
}
