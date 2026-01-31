// Package config provides configuration management for git-trim.
package config

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for git-trim
type Config struct {
	Bases          []string
	Protected      []string
	Update         bool
	UpdateInterval time.Duration
	Confirm        bool
	Detach         bool
	Delete         DeleteFilter
	DryRun         bool
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Bases:          nil,
		Protected:      nil,
		Update:         true,
		UpdateInterval: 5 * time.Second,
		Confirm:        true,
		Detach:         true,
		Delete:         DefaultDeleteFilter(),
		DryRun:         false,
	}
}

// Scope represents the scope for delete operations
type Scope int

const (
	ScopeAll Scope = iota
	ScopeNone
	ScopeScoped
)

// ScopedRemotes holds the scoped remotes
type ScopedRemotes struct {
	Scope   Scope
	Remotes []string
}

// NewScopeAll creates a scope that applies to all remotes
func NewScopeAll() ScopedRemotes {
	return ScopedRemotes{Scope: ScopeAll}
}

// NewScopeNone creates a scope that applies to no remotes
func NewScopeNone() ScopedRemotes {
	return ScopedRemotes{Scope: ScopeNone}
}

// NewScopeScoped creates a scope that applies to specific remotes
func NewScopeScoped(remotes ...string) ScopedRemotes {
	return ScopedRemotes{Scope: ScopeScoped, Remotes: remotes}
}

// Matches returns true if the scope includes the given remote
func (s ScopedRemotes) Matches(remote string) bool {
	switch s.Scope {
	case ScopeAll:
		return true
	case ScopeNone:
		return false
	case ScopeScoped:
		for _, r := range s.Remotes {
			if r == remote || r == "*" {
				return true
			}
		}
		return false
	}
	return false
}

// DeleteFilter specifies which branch types should be deleted
type DeleteFilter struct {
	MergedLocal                     bool
	MergedRemote                    ScopedRemotes
	Stray                           bool
	Diverged                        ScopedRemotes
	MergedNonTrackingLocal          bool
	MergedNonUpstreamRemoteTracking ScopedRemotes
}

// DefaultDeleteFilter returns the default delete filter
func DefaultDeleteFilter() DeleteFilter {
	return DeleteFilter{
		MergedLocal:                     true,
		MergedRemote:                    NewScopeScoped("origin"),
		Stray:                           false,
		Diverged:                        NewScopeNone(),
		MergedNonTrackingLocal:          false,
		MergedNonUpstreamRemoteTracking: NewScopeNone(),
	}
}

// ParseDeleteFilter parses a comma-separated delete filter string
func ParseDeleteFilter(value string) (DeleteFilter, error) {
	filter := DeleteFilter{
		MergedRemote:                    NewScopeNone(),
		Diverged:                        NewScopeNone(),
		MergedNonUpstreamRemoteTracking: NewScopeNone(),
	}

	if value == "" {
		return DefaultDeleteFilter(), nil
	}

	parts := strings.Split(value, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if err := parseDeleteUnit(&filter, part); err != nil {
			return filter, err
		}
	}

	return filter, nil
}

func parseDeleteUnit(filter *DeleteFilter, unit string) error {
	switch {
	case unit == "merged":
		filter.MergedLocal = true
		filter.MergedRemote = NewScopeAll()
	case unit == "merged-local":
		filter.MergedLocal = true
	case strings.HasPrefix(unit, "merged-remote"):
		scope, err := parseScope(unit, "merged-remote")
		if err != nil {
			return err
		}
		filter.MergedRemote = scope
	case unit == "stray":
		filter.Stray = true
	case strings.HasPrefix(unit, "diverged"):
		scope, err := parseScope(unit, "diverged")
		if err != nil {
			return err
		}
		filter.Diverged = scope
	case unit == "local":
		filter.MergedNonTrackingLocal = true
	case strings.HasPrefix(unit, "remote"):
		scope, err := parseScope(unit, "remote")
		if err != nil {
			return err
		}
		filter.MergedNonUpstreamRemoteTracking = scope
	default:
		return fmt.Errorf("unknown delete unit: %s", unit)
	}
	return nil
}

func parseScope(unit, prefix string) (ScopedRemotes, error) {
	if unit == prefix {
		return NewScopeAll(), nil
	}

	if !strings.HasPrefix(unit, prefix+":") {
		return NewScopeNone(), fmt.Errorf("invalid format: %s", unit)
	}

	remote := strings.TrimPrefix(unit, prefix+":")
	if remote == "" {
		return NewScopeNone(), fmt.Errorf("empty remote in: %s", unit)
	}

	if remote == "*" {
		return NewScopeAll(), nil
	}

	return NewScopeScoped(remote), nil
}

// GitConfigReader reads git configuration
type GitConfigReader interface {
	GetString(ctx context.Context, key string) (string, bool)
	GetBool(ctx context.Context, key string) (bool, bool)
	GetInt(ctx context.Context, key string) (int, bool)
}

// LoadFromGitConfig loads configuration from git config
func LoadFromGitConfig(ctx context.Context, reader GitConfigReader, cfg *Config) error {
	if bases, ok := reader.GetString(ctx, "trim.bases"); ok {
		cfg.Bases = parseCommaSeparated(bases)
	}

	if protected, ok := reader.GetString(ctx, "trim.protected"); ok {
		cfg.Protected = parseCommaSeparated(protected)
	}

	if update, ok := reader.GetBool(ctx, "trim.update"); ok {
		cfg.Update = update
	}

	if interval, ok := reader.GetInt(ctx, "trim.updateInterval"); ok {
		cfg.UpdateInterval = time.Duration(interval) * time.Second
	}

	if confirm, ok := reader.GetBool(ctx, "trim.confirm"); ok {
		cfg.Confirm = confirm
	}

	if detach, ok := reader.GetBool(ctx, "trim.detach"); ok {
		cfg.Detach = detach
	}

	if deleteStr, ok := reader.GetString(ctx, "trim.delete"); ok {
		filter, err := ParseDeleteFilter(deleteStr)
		if err != nil {
			return fmt.Errorf("invalid trim.delete config: %w", err)
		}
		cfg.Delete = filter
	}

	return nil
}

func parseCommaSeparated(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// RepoGitConfigReader reads git config via a repository
type RepoGitConfigReader struct {
	RunGit func(ctx context.Context, args ...string) (string, error)
}

func (r *RepoGitConfigReader) GetString(ctx context.Context, key string) (string, bool) {
	value, err := r.RunGit(ctx, "config", "--get", key)
	if err != nil {
		return "", false
	}
	return value, true
}

func (r *RepoGitConfigReader) GetBool(ctx context.Context, key string) (bool, bool) {
	value, err := r.RunGit(ctx, "config", "--get", "--type=bool", key)
	if err != nil {
		return false, false
	}
	parsed, parseErr := strconv.ParseBool(strings.TrimSpace(value))
	if parseErr != nil {
		return false, false
	}
	return parsed, true
}

func (r *RepoGitConfigReader) GetInt(ctx context.Context, key string) (int, bool) {
	value, err := r.RunGit(ctx, "config", "--get", "--type=int", key)
	if err != nil {
		return 0, false
	}
	i, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return i, true
}
