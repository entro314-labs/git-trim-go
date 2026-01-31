package branch_test

import (
	"testing"

	"github.com/dominikospritis/git-trim/internal/branch"
)

func TestLocalBranch(t *testing.T) {
	tests := []struct {
		refname  string
		wantName string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/foo", "feature/foo"},
		{"refs/heads/fix/bar-123", "fix/bar-123"},
	}

	for _, tt := range tests {
		t.Run(tt.refname, func(t *testing.T) {
			lb := branch.LocalBranch{Refname: tt.refname}
			if got := lb.Name(); got != tt.wantName {
				t.Errorf("Name() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

func TestRemoteTrackingBranch(t *testing.T) {
	tests := []struct {
		refname        string
		wantName       string
		wantRemote     string
		wantBranchName string
	}{
		{"refs/remotes/origin/main", "origin/main", "origin", "main"},
		{"refs/remotes/upstream/feature/foo", "upstream/feature/foo", "upstream", "feature/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.refname, func(t *testing.T) {
			rtb := branch.RemoteTrackingBranch{Refname: tt.refname}

			if got := rtb.Name(); got != tt.wantName {
				t.Errorf("Name() = %q, want %q", got, tt.wantName)
			}

			if got := rtb.RemoteName(); got != tt.wantRemote {
				t.Errorf("RemoteName() = %q, want %q", got, tt.wantRemote)
			}

			if got := rtb.BranchName(); got != tt.wantBranchName {
				t.Errorf("BranchName() = %q, want %q", got, tt.wantBranchName)
			}
		})
	}
}

func TestNewLocalBranch(t *testing.T) {
	tests := []struct {
		input   string
		wantRef string
	}{
		{"main", "refs/heads/main"},
		{"refs/heads/main", "refs/heads/main"},
		{"feature/foo", "refs/heads/feature/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lb := branch.NewLocalBranch(tt.input)
			if lb.Refname != tt.wantRef {
				t.Errorf("NewLocalBranch(%q).Refname = %q, want %q", tt.input, lb.Refname, tt.wantRef)
			}
		})
	}
}

func TestFromRemoteTracking(t *testing.T) {
	rtb := branch.RemoteTrackingBranch{Refname: "refs/remotes/origin/feature/foo"}
	rb := branch.FromRemoteTracking(rtb)

	if rb.Remote != "origin" {
		t.Errorf("Remote = %q, want %q", rb.Remote, "origin")
	}

	if rb.Refname != "refs/heads/feature/foo" {
		t.Errorf("Refname = %q, want %q", rb.Refname, "refs/heads/feature/foo")
	}

	if rb.Name() != "feature/foo" {
		t.Errorf("Name() = %q, want %q", rb.Name(), "feature/foo")
	}
}

func TestClassifiedBranchKey(t *testing.T) {
	tests := []struct {
		name   string
		branch branch.ClassifiedBranch
		want   string
	}{
		{
			name:   "MergedLocal",
			branch: branch.MergedLocal{Branch: branch.LocalBranch{Refname: "refs/heads/feature"}},
			want:   "local:refs/heads/feature",
		},
		{
			name:   "Stray",
			branch: branch.Stray{Branch: branch.LocalBranch{Refname: "refs/heads/stray"}},
			want:   "stray:refs/heads/stray",
		},
		{
			name:   "MergedRemoteTracking",
			branch: branch.MergedRemoteTracking{Branch: branch.RemoteTrackingBranch{Refname: "refs/remotes/origin/feature"}},
			want:   "remote-tracking:refs/remotes/origin/feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := branch.ClassifiedBranchKey(tt.branch)
			if got != tt.want {
				t.Errorf("ClassifiedBranchKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
