#!/usr/bin/env bash

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SKIP_TESTS=false
SKIP_SNAPSHOT=false

for arg in "$@"; do
  case "$arg" in
    --skip-tests)
      SKIP_TESTS=true
      ;;
    --skip-snapshot)
      SKIP_SNAPSHOT=true
      ;;
    -h|--help)
      cat <<'EOF'
Usage: ./scripts/release-preflight.sh [OPTIONS]

Options:
  --skip-tests      Skip go test
  --skip-snapshot   Skip goreleaser snapshot build
  -h, --help        Show help
EOF
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $arg${NC}" >&2
      exit 1
      ;;
  esac
done

log() {
  echo -e "${YELLOW}==>${NC} $1"
}

ok() {
  echo -e "${GREEN}✓${NC} $1"
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo -e "${RED}Missing required command: $1${NC}" >&2
    exit 1
  fi
}

need_cmd git
need_cmd go
need_cmd goreleaser

log "Checking git workspace state"
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo -e "${RED}Not inside a git repository${NC}" >&2
  exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo -e "${RED}Working tree is dirty. Commit/stash changes before release.${NC}" >&2
  exit 1
fi
ok "Git workspace is clean"

log "Running dependency checks"
go mod download
go mod verify
ok "Dependencies verified"

if [[ "$SKIP_TESTS" == "false" ]]; then
  log "Running tests"
  go test -v ./...
  ok "Tests passed"
fi

log "Validating GoReleaser configuration"
goreleaser check
ok "GoReleaser configuration valid"

if [[ "$SKIP_SNAPSHOT" == "false" ]]; then
  log "Running snapshot release build (no publish)"
  goreleaser release --snapshot --clean --skip=publish
  ok "Snapshot build completed"
fi

ok "Preflight checks passed. Ready to tag and release."