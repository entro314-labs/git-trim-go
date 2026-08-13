# git-trim

Automatically clean up local Git branches that have been merged or become stray.

## Overview

`git-trim` is a companion tool to `git fetch --prune`. While `git fetch --prune` removes remote-tracking references that no longer exist on the remote, it doesn't clean up the corresponding local branches. `git-trim` fills this gap by:

- Detecting branches that have been fully merged into base branches (master, main, develop)
- Handling multiple merge styles: merge commits (`--no-ff`), rebase/fast-forward, and squash merges
- Cleaning up stray branches (upstream deleted but not merged)
- Supporting various Git workflows: GitHub flow, git-flow, and triangular (fork-based) workflows
- Preserving branches attached to worktrees
- Allowing protected branch patterns via glob syntax
- Running standard Git garbage collection to prune history and optimize repo size
- Converting a repository to a shallow clone (depth=1) to save maximum history space

## Installation

### From Source

```bash
go install github.com/dominikospritis/git-trim/cmd/git-trim@latest
```

### From Releases

Download the appropriate binary from the [releases page](https://github.com/dominikospritis/git-trim/releases).

## Usage

```bash
# Preview what would be deleted (dry run)
git-trim --dry-run

# Delete merged branches
git-trim

# Specify base branches explicitly
git-trim --bases main,develop

# Protect branches matching patterns
git-trim --protected "feature/*,release/*"

# Delete stray branches (upstream gone, not merged)
git-trim --delete stray

# Skip confirmation prompt
git-trim --no-confirm

# Skip remote update
git-trim --no-update

# Run Git garbage collection after trimming to prune history
git-trim --gc

# Truncate the entire local history to a shallow clone to aggressively save space
git-trim --shallow
```

## Configuration

git-trim can be configured via command-line flags or git config:

| Flag | Git Config | Description | Default |
|------|------------|-------------|---------|
| `--bases` | `trim.bases` | Comma-separated base branches | auto-detect |
| `--protected` | `trim.protected` | Glob patterns to protect | none |
| `--update/--no-update` | `trim.update` | Update remotes before trimming | true |
| `--update-interval` | `trim.updateInterval` | Seconds between updates | 5 |
| `--confirm/--no-confirm` | `trim.confirm` | Require confirmation | true |
| `--detach/--no-detach` | `trim.detach` | Allow HEAD detachment | true |
| `--delete` | `trim.delete` | Delete ranges | merged:origin |
| `--gc` | `trim.gc` | Run aggressive GC after trimming | false |
| `--shallow` | `trim.shallow` | Truncate history to depth 1 | false |

### Delete Ranges

The `--delete` flag accepts comma-separated values:

- `merged` - Shorthand for `merged-local,merged-remote`
- `merged-local` - Delete local branches tracking merged upstreams
- `merged-remote` - Delete merged remote branches (use `merged-remote:origin`)
- `stray` - Delete local branches with gone upstreams (not merged)
- `diverged:<remote>` - Delete local + diverged upstream branches
- `local` - Delete merged non-tracking local branches
- `remote:<remote>` - Delete merged orphan remote-tracking branches

Use `*` as remote name to target all remotes.

## Branch Classification

git-trim classifies branches into the following categories:

| Classification | Description |
|----------------|-------------|
| `MergedLocal` | Local branch fully merged into base |
| `Stray` | Upstream deleted, local not merged (dangerous!) |
| `MergedRemoteTracking` | Remote-tracking ref fully merged |
| `DivergedRemoteTracking` | Local merged but diverged from upstream |
| `MergedNonTrackingLocal` | Non-tracking local branch merged |
| `MergedNonUpstreamRemoteTracking` | Orphan remote-tracking ref merged |

## Merge Detection

git-trim detects three merge styles:

1. **No-fast-forward merge** (`--no-ff`) - Merge commit preserves branch history
2. **Rebase/Fast-forward** - Commits rebased onto base branch
3. **Squash merge** - All commits squashed into single commit

## Workflow Support

### GitHub Flow
Standard feature branch workflow with `main` as the base branch.

### Git Flow
Supports `master` and `develop` as base branches with feature/release branches.

### Triangular (Fork-based)
Supports fork workflows where you push to your fork but merge into upstream.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Credits

This is a Go rewrite inspired by the original [git-trim](https://github.com/foriequal0/git-trim) written in Rust.
