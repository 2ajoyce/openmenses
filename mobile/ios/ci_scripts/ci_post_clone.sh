#!/bin/bash
set -euo pipefail

# ci_post_clone.sh — Xcode Cloud runs this after cloning the repo.
# It builds Engine.xcframework (gomobile bind) and the UI bundle (Vite)
# so the Xcode archive step has everything it needs.
#
# Xcode Cloud provides Homebrew pre-installed; no sudo is available.

REPO_ROOT="${CI_PRIMARY_REPOSITORY_PATH:-$(cd "$(dirname "$0")/../../.." && pwd)}"
cd "$REPO_ROOT"

# retry — retry a command up to N times with exponential backoff.
# Usage: retry <max_attempts> <command...>
retry() {
  local max_attempts=$1; shift
  local attempt=1
  local delay=5
  while true; do
    echo "  Attempt $attempt/$max_attempts: $*"
    if "$@"; then
      return 0
    fi
    if (( attempt >= max_attempts )); then
      echo "  Failed after $max_attempts attempts."
      return 1
    fi
    echo "  Retrying in ${delay}s..."
    sleep "$delay"
    attempt=$((attempt + 1))
    delay=$((delay * 2))
  done
}

# --- Install Go ---
echo "--- Installing Go @1.25 ---"
brew install go@1.25
export PATH="$(brew --prefix go@1.25)/bin:$(go env GOPATH)/bin:$PATH"
go version

# --- Install gomobile ---
echo "--- Installing gomobile ---"
retry 3 go install golang.org/x/mobile/cmd/gomobile@latest
retry 3 go install golang.org/x/mobile/cmd/gobind@latest
PATH="$(go env GOPATH)/bin:$PATH" gomobile init

# --- Build Engine.xcframework ---
echo "--- Building Engine.xcframework ---"
make mobile-ios

# --- Install Node.js ---
echo "--- Installing Node.js ---"
if ! command -v node &>/dev/null; then
  brew install node@22
  export PATH="$(brew --prefix node@22)/bin:$PATH"
fi
export PATH="$(brew --prefix node@22)/bin:$PATH"
node --version
npm --version

# --- Build UI bundle ---
echo "--- Building UI bundle ---"
retry 3 bash -c 'cd ui && npm ci'
make ui-bundle

echo "--- ci_post_clone.sh complete ---"
