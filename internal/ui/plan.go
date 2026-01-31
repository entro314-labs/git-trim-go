package ui

import (
	"fmt"

	"github.com/dominikospritis/git-trim/internal/branch"
	"github.com/dominikospritis/git-trim/internal/core"
)

// DisplayPlan displays the trim plan in a formatted way
func DisplayPlan(plan *core.TrimPlan, dryRun bool) {
	if plan.IsEmpty() {
		PrintNoChanges()
		return
	}

	if dryRun {
		PrintDryRun()
	}

	localBranches := make([]branch.ClassifiedBranch, 0)
	remoteBranches := make([]branch.ClassifiedBranch, 0)
	strayBranches := make([]branch.ClassifiedBranch, 0)

	for _, cb := range plan.ToDelete {
		switch cb.Type() {
		case branch.TypeStray:
			strayBranches = append(strayBranches, cb)
		case branch.TypeMergedRemoteTracking, branch.TypeMergedNonUpstreamRemoteTracking:
			remoteBranches = append(remoteBranches, cb)
		default:
			if cb.LocalBranch() != nil {
				localBranches = append(localBranches, cb)
			}
		}
	}

	if len(localBranches) > 0 {
		PrintSubHeader("Local branches to delete")
		for _, cb := range localBranches {
			displayClassifiedBranch(cb)
		}
	}

	if len(remoteBranches) > 0 {
		PrintSubHeader("Remote branches to delete")
		for _, cb := range remoteBranches {
			displayClassifiedBranch(cb)
		}
	}

	if len(strayBranches) > 0 {
		PrintSubHeader("Stray branches (upstream gone, not merged)")
		ColorWarning.Println("  These branches have unmerged commits!")
		for _, cb := range strayBranches {
			displayClassifiedBranch(cb)
		}
	}

	if len(plan.Preserved) > 0 {
		PrintSubHeader("Preserved branches")
		for _, p := range plan.Preserved {
			displayPreserved(p)
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d branch(es) to delete", len(plan.ToDelete))
	if len(plan.Preserved) > 0 {
		fmt.Printf(", %d preserved", len(plan.Preserved))
	}
	fmt.Println()
}

func displayClassifiedBranch(cb branch.ClassifiedBranch) {
	var icon, name, detail string

	switch v := cb.(type) {
	case branch.MergedLocal:
		icon = "├─"
		name = ColorBranch.Sprint(v.Branch.Name())
		detail = ColorSuccess.Sprint("merged")

	case branch.Stray:
		icon = "├─"
		name = ColorBranch.Sprint(v.Branch.Name())
		detail = ColorWarning.Sprint("stray (not merged!)")

	case branch.MergedRemoteTracking:
		icon = "├─"
		name = ColorRemote.Sprint(v.Branch.RemoteName()) + "/" + ColorBranch.Sprint(v.Branch.BranchName())
		detail = ColorSuccess.Sprint("merged")

	case branch.DivergedRemoteTracking:
		icon = "├─"
		name = ColorBranch.Sprint(v.Local.Name()) + " + " + ColorRemote.Sprint(v.Upstream.Name())
		detail = ColorWarning.Sprint("diverged")

	case branch.MergedDirectFetch:
		icon = "├─"
		name = ColorBranch.Sprint(v.Local.Name())
		detail = ColorSuccess.Sprint("direct-fetch merged")

	case branch.DivergedDirectFetch:
		icon = "├─"
		name = ColorBranch.Sprint(v.Local.Name())
		detail = ColorWarning.Sprint("direct-fetch diverged")

	case branch.MergedNonTrackingLocal:
		icon = "├─"
		name = ColorBranch.Sprint(v.Branch.Name())
		detail = ColorSuccess.Sprint("non-tracking merged")

	case branch.MergedNonUpstreamRemoteTracking:
		icon = "├─"
		name = ColorRemote.Sprint(v.Branch.RemoteName()) + "/" + ColorBranch.Sprint(v.Branch.BranchName())
		detail = ColorPreserved.Sprint("orphan merged")

	default:
		icon = "├─"
		name = cb.Description()
		detail = ""
	}

	if detail != "" {
		fmt.Printf("  %s %s (%s)\n", icon, name, detail)
	} else {
		fmt.Printf("  %s %s\n", icon, name)
	}
}

func displayPreserved(p core.Preserved) {
	var name string
	if lb := p.Branch.LocalBranch(); lb != nil {
		name = ColorPreserved.Sprint(lb.Name())
	} else if rtb := p.Branch.RemoteTrackingBranch(); rtb != nil {
		name = ColorPreserved.Sprint(rtb.Name())
	} else {
		name = ColorPreserved.Sprint(p.Branch.Description())
	}

	reason := ColorPreserved.Sprint(p.Reason.String())
	if p.Details != "" {
		reason += ColorPreserved.Sprintf(" @ %s", p.Details)
	}

	fmt.Printf("  ├─ %s (%s)\n", name, reason)
}

// DisplayResults displays the results of execution
func DisplayResults(results []core.DeleteResult) {
	fmt.Println()
	PrintSubHeader("Results")

	successCount := 0
	failCount := 0

	for _, r := range results {
		if r.Success {
			successCount++
			ColorSuccess.Printf("  ✓ Deleted %s\n", r.Branch)
		} else {
			failCount++
			ColorError.Printf("  ✗ Failed to delete %s: %v\n", r.Branch, r.Error)
		}
	}

	fmt.Println()
	if failCount == 0 {
		ColorSuccess.Printf("Successfully deleted %d branch(es)\n", successCount)
	} else {
		ColorWarning.Printf("Deleted %d branch(es), %d failed\n", successCount, failCount)
	}
}
