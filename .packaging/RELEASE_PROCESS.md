# git-trim Release and Packaging Process

This document describes the release pipeline configured in this repository.

## Automated release flow

The pipeline uses Release Please + GoReleaser v2 on GitHub Actions:

1. `Release Please` updates/opens a release PR from conventional commits.
2. Merging the release PR creates a semantic tag (`vX.Y.Z`).
3. The `Release` workflow runs GoReleaser.
4. Artifacts are published to GitHub Releases.
5. Package managers are updated automatically (when secrets are configured):
   - Homebrew Cask (`entro314-labs/homebrew-tap`)
   - AUR (`git-trim-bin`)
   - Winget (`microsoft/winget-pkgs`, via PR)

## Required repository secrets

Set these in GitHub Actions secrets before enabling publish mode:

- `HOMEBREW_TAP_GITHUB_TOKEN`
- `AUR_SSH_PRIVATE_KEY`
- `WINGET_GITHUB_TOKEN`

`GITHUB_TOKEN` is provided automatically by GitHub Actions.

## Local preflight

Use helper scripts before tagging:

- `./scripts/release-preflight.sh`
- `./scripts/create-release-tag.sh v1.2.3`

## Core files

- `.goreleaser.yml` - canonical GoReleaser configuration
- `.github/workflows/ci.yml` - CI checks and snapshot validation
- `.github/workflows/release.yml` - tagged release execution
- `.github/workflows/release-please.yml` - automated release PR/tag creation
- `.release-please-config.json` / `.release-please-manifest.json` - Release Please configuration

## Notes

- Homebrew uses `homebrew_casks` (not deprecated `brews`).
- Winget and Homebrew push to external repositories, so PAT-based secrets are required.
- AUR publishing requires SSH key access to the package repo.