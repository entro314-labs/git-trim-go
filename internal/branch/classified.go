package branch

import "fmt"

// ClassificationType represents the type of branch classification
type ClassificationType int

const (
	TypeMergedLocal ClassificationType = iota
	TypeStray
	TypeMergedRemoteTracking
	TypeDivergedRemoteTracking
	TypeMergedDirectFetch
	TypeDivergedDirectFetch
	TypeMergedNonTrackingLocal
	TypeMergedNonUpstreamRemoteTracking
)

// ClassifiedBranch represents a branch that has been classified for potential deletion
type ClassifiedBranch interface {
	Type() ClassificationType
	Description() string
	LocalBranch() *LocalBranch
	RemoteTrackingBranch() *RemoteTrackingBranch
	RemoteBranch() *RemoteBranch
}

// MergedLocal represents a local branch that has been merged
type MergedLocal struct {
	Branch LocalBranch
}

func (m MergedLocal) Type() ClassificationType                        { return TypeMergedLocal }
func (m MergedLocal) Description() string                             { return fmt.Sprintf("merged local: %s", m.Branch.Name()) }
func (m MergedLocal) LocalBranch() *LocalBranch                       { return &m.Branch }
func (m MergedLocal) RemoteTrackingBranch() *RemoteTrackingBranch     { return nil }
func (m MergedLocal) RemoteBranch() *RemoteBranch                     { return nil }

// Stray represents a local branch whose upstream is gone but is not merged
type Stray struct {
	Branch LocalBranch
}

func (s Stray) Type() ClassificationType                        { return TypeStray }
func (s Stray) Description() string                             { return fmt.Sprintf("stray (upstream gone, not merged): %s", s.Branch.Name()) }
func (s Stray) LocalBranch() *LocalBranch                       { return &s.Branch }
func (s Stray) RemoteTrackingBranch() *RemoteTrackingBranch     { return nil }
func (s Stray) RemoteBranch() *RemoteBranch                     { return nil }

// MergedRemoteTracking represents a remote-tracking branch that has been merged
type MergedRemoteTracking struct {
	Branch RemoteTrackingBranch
}

func (m MergedRemoteTracking) Type() ClassificationType                        { return TypeMergedRemoteTracking }
func (m MergedRemoteTracking) Description() string                             { return fmt.Sprintf("merged remote: %s", m.Branch.Name()) }
func (m MergedRemoteTracking) LocalBranch() *LocalBranch                       { return nil }
func (m MergedRemoteTracking) RemoteTrackingBranch() *RemoteTrackingBranch     { return &m.Branch }
func (m MergedRemoteTracking) RemoteBranch() *RemoteBranch {
	rb := FromRemoteTracking(m.Branch)
	return &rb
}

// DivergedRemoteTracking represents a local branch that is merged but diverged from upstream
type DivergedRemoteTracking struct {
	Local    LocalBranch
	Upstream RemoteTrackingBranch
}

func (d DivergedRemoteTracking) Type() ClassificationType { return TypeDivergedRemoteTracking }
func (d DivergedRemoteTracking) Description() string {
	return fmt.Sprintf("diverged: local %s merged, upstream %s diverged", d.Local.Name(), d.Upstream.Name())
}
func (d DivergedRemoteTracking) LocalBranch() *LocalBranch                       { return &d.Local }
func (d DivergedRemoteTracking) RemoteTrackingBranch() *RemoteTrackingBranch     { return &d.Upstream }
func (d DivergedRemoteTracking) RemoteBranch() *RemoteBranch {
	rb := FromRemoteTracking(d.Upstream)
	return &rb
}

// MergedDirectFetch represents a hub-cli style branch that is merged
type MergedDirectFetch struct {
	Local  LocalBranch
	Remote RemoteBranch
}

func (m MergedDirectFetch) Type() ClassificationType                        { return TypeMergedDirectFetch }
func (m MergedDirectFetch) Description() string                             { return fmt.Sprintf("merged direct-fetch: %s -> %s", m.Local.Name(), m.Remote.String()) }
func (m MergedDirectFetch) LocalBranch() *LocalBranch                       { return &m.Local }
func (m MergedDirectFetch) RemoteTrackingBranch() *RemoteTrackingBranch     { return nil }
func (m MergedDirectFetch) RemoteBranch() *RemoteBranch                     { return &m.Remote }

// DivergedDirectFetch represents a hub-cli style branch where local is merged but remote diverged
type DivergedDirectFetch struct {
	Local  LocalBranch
	Remote RemoteBranch
}

func (d DivergedDirectFetch) Type() ClassificationType { return TypeDivergedDirectFetch }
func (d DivergedDirectFetch) Description() string {
	return fmt.Sprintf("diverged direct-fetch: local %s merged, remote %s diverged", d.Local.Name(), d.Remote.String())
}
func (d DivergedDirectFetch) LocalBranch() *LocalBranch                       { return &d.Local }
func (d DivergedDirectFetch) RemoteTrackingBranch() *RemoteTrackingBranch     { return nil }
func (d DivergedDirectFetch) RemoteBranch() *RemoteBranch                     { return &d.Remote }

// MergedNonTrackingLocal represents a non-tracking local branch that is merged
type MergedNonTrackingLocal struct {
	Branch LocalBranch
}

func (m MergedNonTrackingLocal) Type() ClassificationType                        { return TypeMergedNonTrackingLocal }
func (m MergedNonTrackingLocal) Description() string                             { return fmt.Sprintf("merged non-tracking local: %s", m.Branch.Name()) }
func (m MergedNonTrackingLocal) LocalBranch() *LocalBranch                       { return &m.Branch }
func (m MergedNonTrackingLocal) RemoteTrackingBranch() *RemoteTrackingBranch     { return nil }
func (m MergedNonTrackingLocal) RemoteBranch() *RemoteBranch                     { return nil }

// MergedNonUpstreamRemoteTracking represents an orphan remote-tracking branch that is merged
type MergedNonUpstreamRemoteTracking struct {
	Branch RemoteTrackingBranch
}

func (m MergedNonUpstreamRemoteTracking) Type() ClassificationType { return TypeMergedNonUpstreamRemoteTracking }
func (m MergedNonUpstreamRemoteTracking) Description() string {
	return fmt.Sprintf("merged orphan remote-tracking: %s", m.Branch.Name())
}
func (m MergedNonUpstreamRemoteTracking) LocalBranch() *LocalBranch                       { return nil }
func (m MergedNonUpstreamRemoteTracking) RemoteTrackingBranch() *RemoteTrackingBranch     { return &m.Branch }
func (m MergedNonUpstreamRemoteTracking) RemoteBranch() *RemoteBranch {
	rb := FromRemoteTracking(m.Branch)
	return &rb
}

// ClassifiedBranchKey generates a unique key for deduplication
func ClassifiedBranchKey(cb ClassifiedBranch) string {
	switch v := cb.(type) {
	case MergedLocal:
		return "local:" + v.Branch.Refname
	case Stray:
		return "stray:" + v.Branch.Refname
	case MergedRemoteTracking:
		return "remote-tracking:" + v.Branch.Refname
	case DivergedRemoteTracking:
		return "diverged:" + v.Local.Refname + ":" + v.Upstream.Refname
	case MergedDirectFetch:
		return "direct-fetch:" + v.Local.Refname
	case DivergedDirectFetch:
		return "diverged-direct:" + v.Local.Refname
	case MergedNonTrackingLocal:
		return "non-tracking:" + v.Branch.Refname
	case MergedNonUpstreamRemoteTracking:
		return "non-upstream:" + v.Branch.Refname
	default:
		return fmt.Sprintf("unknown:%T", cb)
	}
}
