#!/bin/bash
set -euo pipefail

# ci_post_clone.sh — Xcode Cloud runs this after cloning the repo.
# It builds Engine.xcframework (gomobile bind) and the UI bundle (Vite)
# so the Xcode archive step has everything it needs.

REPO_ROOT="${CI_PRIMARY_REPOSITORY_PATH:-$(cd "$(dirname "$0")/../../.." && pwd)}"
cd "$REPO_ROOT"

GO_VERSION="1.25.0"
NODE_VERSION="22"

# --- Install Go ---
echo "--- Installing Go ${GO_VERSION} ---"
curl -fsSL "https://go.dev/dl/go${GO_VERSION}.darwin-arm64.tar.gz" -o /tmp/go.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz
export PATH="/usr/local/go/bin:$(go env GOPATH)/bin:$PATH"
go version

# --- Install gomobile ---
echo "--- Installing gomobile ---"
make mobile-setup

# --- Build Engine.xcframework ---
echo "--- Building Engine.xcframework ---"
make mobile-ios

# --- Install Node.js ---
echo "--- Installing Node.js ${NODE_VERSION} ---"
if ! command -v node &>/dev/null; then
  curl -fsSL "https://nodejs.org/dist/latest-v${NODE_VERSION}.x/SHASUMS256.txt" -o /tmp/node-shasums.txt
  NODE_FILENAME=$(grep "darwin-arm64.tar.gz" /tmp/node-shasums.txt | awk '{print $2}')
  NODE_URL="https://nodejs.org/dist/latest-v${NODE_VERSION}.x/${NODE_FILENAME}"
  curl -fsSL "$NODE_URL" -o /tmp/node.tar.gz
  sudo mkdir -p /usr/local/node
  sudo tar -C /usr/local/node --strip-components=1 -xzf /tmp/node.tar.gz
  rm /tmp/node.tar.gz /tmp/node-shasums.txt
  export PATH="/usr/local/node/bin:$PATH"
fi
node --version
npm --version

# --- Build UI bundle ---
echo "--- Building UI bundle ---"
cd ui && npm ci && cd ..
make ui-bundle

echo "--- ci_post_clone.sh complete ---"
