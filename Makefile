# Root Makefile — openmenses
#
# Use these targets for all development and CI tasks.
# Do not run underlying tools directly unless debugging a specific issue.

.PHONY: proto-lint proto-generate proto-breaking \
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
	cd ui && npm run lint && npm run typecheck

ui-test:
	cd ui && npm run test

# ---------------------------------------------------------------------------
# Aggregate targets
# ---------------------------------------------------------------------------

# Run all linters
lint: proto-lint engine-lint ui-lint

# Run all tests
test: engine-test ui-test

# Full CI validation (linting + generation check + tests)
ci: proto-lint proto-generate engine-lint engine-test ui-lint ui-test
