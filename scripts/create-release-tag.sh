#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: ./scripts/create-release-tag.sh vMAJOR.MINOR.PATCH [--no-push]"
  exit 1
fi

TAG="$1"
NO_PUSH=false

if [[ "${2:-}" == "--no-push" ]]; then
  NO_PUSH=true
fi

if [[ ! "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$ ]]; then
  echo "Invalid tag format: $TAG"
  echo "Expected: vMAJOR.MINOR.PATCH (optionally with prerelease/build metadata)"
  exit 1
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Not inside a git repository"
  exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "Working tree is dirty. Commit/stash changes before tagging."
  exit 1
fi

if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "Tag already exists: $TAG"
  exit 1
fi

BRANCH="$(git rev-parse --abbrev-ref HEAD)"

git tag -a "$TAG" -m "Release $TAG"
echo "Created tag $TAG on branch $BRANCH"

if [[ "$NO_PUSH" == "false" ]]; then
  git push origin "$BRANCH"
  git push origin "$TAG"
  echo "Pushed branch and tag to origin"
else
  echo "Tag created locally only (--no-push)"
fi