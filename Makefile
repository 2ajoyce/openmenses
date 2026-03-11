# Root Makefile — openmenses
#
# Use these targets for all development and CI tasks.
# Do not run underlying tools directly unless debugging a specific issue.

# On Windows, npm is npm.cmd; on Unix it is npm.
ifeq ($(OS),Windows_NT)
    NPM := npm.cmd
else
    NPM := npm
endif

.PHONY: proto-lint proto-generate proto-breaking proto-check \
        engine-lint engine-test \
        ui-lint ui-test \
        lint test ci

# ---------------------------------------------------------------------------
# Proto targets (requires buf: https://buf.build/docs/installation)
# ---------------------------------------------------------------------------

proto-lint:
	buf lint

proto-generate:
	buf generate

# Run buf generate and fail if gen/ differs from what is committed.
# Use this in CI to catch proto changes whose generated output was not committed.
proto-check:
	buf generate
	git diff --exit-code -- gen/

proto-breaking:
	buf breaking --against '.git#branch=main'

# ---------------------------------------------------------------------------
# Engine targets (Go domain engine)
# ---------------------------------------------------------------------------

engine-lint:
	test -z "$$(gofmt -l engine/)" && go vet ./engine/... && golangci-lint run ./engine/...

engine-test:
	go test ./engine/...

# ---------------------------------------------------------------------------
# UI targets (TypeScript)
# ---------------------------------------------------------------------------

ui-lint:
	cd ui && $(NPM) run lint && $(NPM) run typecheck

ui-test:
	cd ui && $(NPM) run test

# ---------------------------------------------------------------------------
# Aggregate targets
# ---------------------------------------------------------------------------

# Run all linters
lint: proto-lint engine-lint ui-lint

# Run all tests
test: engine-test ui-test

# Full CI validation (linting + generation check + tests)
ci: proto-lint proto-check engine-lint engine-test ui-lint ui-test
