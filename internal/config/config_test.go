package config_test

import (
	"context"
	"testing"

	"github.com/dominikospritis/git-trim/internal/config"
)

func TestParseDeleteFilter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		check   func(t *testing.T, f config.DeleteFilter)
		wantErr bool
	}{
		{
			name:  "empty uses default",
			input: "",
			check: func(t *testing.T, f config.DeleteFilter) {
				if !f.MergedLocal {
					t.Error("expected MergedLocal to be true")
				}
			},
		},
		{
			name:  "merged enables all",
			input: "merged",
			check: func(t *testing.T, f config.DeleteFilter) {
				if !f.MergedLocal {
					t.Error("expected MergedLocal to be true")
				}
				if !f.MergedRemote.Matches("origin") {
					t.Error("expected MergedRemote to match origin")
				}
				if !f.MergedRemote.Matches("upstream") {
					t.Error("expected MergedRemote to match upstream")
				}
			},
		},
		{
			name:  "merged-local only",
			input: "merged-local",
			check: func(t *testing.T, f config.DeleteFilter) {
				if !f.MergedLocal {
					t.Error("expected MergedLocal to be true")
				}
				if f.MergedRemote.Matches("origin") {
					t.Error("expected MergedRemote to not match")
				}
			},
		},
		{
			name:  "merged-remote scoped",
			input: "merged-remote:origin",
			check: func(t *testing.T, f config.DeleteFilter) {
				if !f.MergedRemote.Matches("origin") {
					t.Error("expected MergedRemote to match origin")
				}
				if f.MergedRemote.Matches("upstream") {
					t.Error("expected MergedRemote to not match upstream")
				}
			},
		},
		{
			name:  "stray",
			input: "stray",
			check: func(t *testing.T, f config.DeleteFilter) {
				if !f.Stray {
					t.Error("expected Stray to be true")
				}
			},
		},
		{
			name:  "diverged scoped",
			input: "diverged:origin",
			check: func(t *testing.T, f config.DeleteFilter) {
				if !f.Diverged.Matches("origin") {
					t.Error("expected Diverged to match origin")
				}
			},
		},
		{
			name:  "local non-tracking",
			input: "local",
			check: func(t *testing.T, f config.DeleteFilter) {
				if !f.MergedNonTrackingLocal {
					t.Error("expected MergedNonTrackingLocal to be true")
				}
			},
		},
		{
			name:  "remote orphan",
			input: "remote:origin",
			check: func(t *testing.T, f config.DeleteFilter) {
				if !f.MergedNonUpstreamRemoteTracking.Matches("origin") {
					t.Error("expected MergedNonUpstreamRemoteTracking to match origin")
				}
			},
		},
		{
			name:  "multiple",
			input: "merged-local,stray,diverged:origin",
			check: func(t *testing.T, f config.DeleteFilter) {
				if !f.MergedLocal {
					t.Error("expected MergedLocal")
				}
				if !f.Stray {
					t.Error("expected Stray")
				}
				if !f.Diverged.Matches("origin") {
					t.Error("expected Diverged")
				}
			},
		},
		{
			name:    "unknown unit",
			input:   "unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := config.ParseDeleteFilter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDeleteFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, f)
			}
		})
	}
}

func TestGlobMatcher(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		branch   string
		want     bool
	}{
		{
			name:     "exact match",
			patterns: []string{"main"},
			branch:   "main",
			want:     true,
		},
		{
			name:     "no match",
			patterns: []string{"main"},
			branch:   "develop",
			want:     false,
		},
		{
			name:     "wildcard",
			patterns: []string{"feature/*"},
			branch:   "feature/foo",
			want:     true,
		},
		{
			name:     "double star",
			patterns: []string{"feature/**"},
			branch:   "feature/foo/bar",
			want:     true,
		},
		{
			name:     "multiple patterns",
			patterns: []string{"main", "develop", "release/*"},
			branch:   "release/1.0",
			want:     true,
		},
		{
			name:     "refs/heads prefix stripped",
			patterns: []string{"main"},
			branch:   "refs/heads/main",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := config.NewGlobMatcher(tt.patterns)
			got := m.Matches(tt.branch)
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}
}

func TestScopedRemotes(t *testing.T) {
	tests := []struct {
		name   string
		scope  config.ScopedRemotes
		remote string
		want   bool
	}{
		{"all matches everything", config.NewScopeAll(), "origin", true},
		{"all matches anything", config.NewScopeAll(), "upstream", true},
		{"none matches nothing", config.NewScopeNone(), "origin", false},
		{"scoped matches exact", config.NewScopeScoped("origin"), "origin", true},
		{"scoped no match", config.NewScopeScoped("origin"), "upstream", false},
		{"scoped wildcard", config.NewScopeScoped("*"), "anything", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scope.Matches(tt.remote)
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.remote, got, tt.want)
			}
		})
	}
}

func TestRepoGitConfigReader_GetBoolParsesValues(t *testing.T) {
	reader := config.RepoGitConfigReader{
		RunGit: func(ctx context.Context, args ...string) (string, error) {
			return " false ", nil
		},
	}

	val, ok := reader.GetBool(context.Background(), "trim.update")
	if !ok {
		t.Fatalf("expected ok for parsed bool")
	}
	if val {
		t.Fatalf("expected parsed bool to be false")
	}
}

func TestRepoGitConfigReader_GetStringTreatsEmptyAsSet(t *testing.T) {
	reader := config.RepoGitConfigReader{
		RunGit: func(ctx context.Context, args ...string) (string, error) {
			return "", nil
		},
	}

	val, ok := reader.GetString(context.Background(), "trim.protected")
	if !ok {
		t.Fatalf("expected ok even for empty value")
	}
	if val != "" {
		t.Fatalf("expected empty string to be returned as-is")
	}
}

func TestRepoGitConfigReader_GetIntTrimsWhitespace(t *testing.T) {
	reader := config.RepoGitConfigReader{
		RunGit: func(ctx context.Context, args ...string) (string, error) {
			return " 7 ", nil
		},
	}

	val, ok := reader.GetInt(context.Background(), "trim.updateInterval")
	if !ok {
		t.Fatalf("expected ok for parsed int")
	}
	if val != 7 {
		t.Fatalf("expected int to parse trimmed value, got %d", val)
	}
}
