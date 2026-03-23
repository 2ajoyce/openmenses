#!/bin/bash
set -euo pipefail

# ci_post_clone.sh — Xcode Cloud runs this after cloning the repo.
# It builds Engine.xcframework (gomobile bind) and the UI bundle (Vite)
# so the Xcode archive step has everything it needs.
#
# Xcode Cloud provides Homebrew pre-installed; no sudo is available.

REPO_ROOT="${CI_PRIMARY_REPOSITORY_PATH:-$(cd "$(dirname "$0")/../../.." && pwd)}"
cd "$REPO_ROOT"

# --- Install Go ---
echo "--- Installing Go @1.25 ---"
brew install go@1.25
export PATH="$(brew --prefix go@1.25)/bin:$(go env GOPATH)/bin:$PATH"
go version

# --- Install gomobile ---
echo "--- Installing gomobile ---"
make mobile-setup

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
cd ui && npm ci && cd ..
make ui-bundle

echo "--- ci_post_clone.sh complete ---"
