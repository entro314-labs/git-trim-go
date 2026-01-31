// Package branch provides types and utilities for Git branch representation.
package branch

import (
	"fmt"
	"strings"
)

// LocalBranch represents a local Git branch (refs/heads/*)
type LocalBranch struct {
	Refname string // Full ref name, e.g., "refs/heads/feature/foo"
}

// Name returns the short name of the branch (without refs/heads/ prefix)
func (b LocalBranch) Name() string {
	return strings.TrimPrefix(b.Refname, "refs/heads/")
}

// String returns a human-readable representation
func (b LocalBranch) String() string {
	return b.Name()
}

// RemoteTrackingBranch represents a remote-tracking branch (refs/remotes/*/*)
type RemoteTrackingBranch struct {
	Refname string // Full ref name, e.g., "refs/remotes/origin/main"
}

// Name returns the short name (without refs/remotes/ prefix)
func (b RemoteTrackingBranch) Name() string {
	return strings.TrimPrefix(b.Refname, "refs/remotes/")
}

// RemoteName returns the remote name (e.g., "origin")
func (b RemoteTrackingBranch) RemoteName() string {
	name := b.Name()
	parts := strings.SplitN(name, "/", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// BranchName returns just the branch name without remote prefix
func (b RemoteTrackingBranch) BranchName() string {
	name := b.Name()
	parts := strings.SplitN(name, "/", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return name
}

// String returns a human-readable representation
func (b RemoteTrackingBranch) String() string {
	return b.Name()
}

// RemoteBranch represents a branch on a remote (for push --delete operations)
type RemoteBranch struct {
	Remote  string // Remote name or URL, e.g., "origin"
	Refname string // Branch ref on remote, e.g., "refs/heads/feature/foo"
}

// Name returns the short branch name
func (b RemoteBranch) Name() string {
	return strings.TrimPrefix(b.Refname, "refs/heads/")
}

// String returns a human-readable representation
func (b RemoteBranch) String() string {
	return fmt.Sprintf("%s/%s", b.Remote, b.Name())
}

// TrackingBranch holds a local branch with its upstream remote-tracking branch
type TrackingBranch struct {
	Local    LocalBranch
	Upstream RemoteTrackingBranch
}

// DirectFetchBranch represents a branch configured with hub-cli style direct fetch
type DirectFetchBranch struct {
	Local  LocalBranch
	Remote RemoteBranch
}

// NewLocalBranch creates a LocalBranch from a ref name
func NewLocalBranch(refname string) LocalBranch {
	if !strings.HasPrefix(refname, "refs/heads/") {
		refname = "refs/heads/" + refname
	}
	return LocalBranch{Refname: refname}
}

// NewRemoteTrackingBranch creates a RemoteTrackingBranch from a ref name
func NewRemoteTrackingBranch(refname string) RemoteTrackingBranch {
	if !strings.HasPrefix(refname, "refs/remotes/") {
		refname = "refs/remotes/" + refname
	}
	return RemoteTrackingBranch{Refname: refname}
}

// NewRemoteBranch creates a RemoteBranch
func NewRemoteBranch(remote, refname string) RemoteBranch {
	if !strings.HasPrefix(refname, "refs/heads/") {
		refname = "refs/heads/" + refname
	}
	return RemoteBranch{Remote: remote, Refname: refname}
}

// FromRemoteTracking creates a RemoteBranch from a RemoteTrackingBranch
func FromRemoteTracking(rtb RemoteTrackingBranch) RemoteBranch {
	return RemoteBranch{
		Remote:  rtb.RemoteName(),
		Refname: "refs/heads/" + rtb.BranchName(),
	}
}
