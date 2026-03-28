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
        engine-dev ui-dev ui-build ui-lint ui-test \
        seed seed-all fixtures-generate \
        mobile-setup ui-bundle mobile-ios mobile-project \
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

# Development: run engine-dev and Vite dev server in separate terminals.
#   Terminal 1: make engine-dev
#   Terminal 2: make ui-dev
# Pass DB= to use a SQLite backend: make engine-dev DB=openmenses-regular-12.db
DB ?=

engine-dev:
	go run ./engine/cmd/engine-dev --port 8080 $(if $(DB),--db=$(DB),)

ui-dev:
	cd ui && $(NPM) run dev

ui-build:
	cd ui && $(NPM) run build

ui-lint:
	cd ui && $(NPM) run lint && $(NPM) run typecheck

ui-test:
	cd ui && $(NPM) run test

# ---------------------------------------------------------------------------
# Seed Data targets (Generate realistic test data for manual testing)
# ---------------------------------------------------------------------------

SCENARIO ?= regular-12
CYCLES   ?= 0
SEED     ?= 42

# Generate a single scenario (default: regular-12) into openmenses.db.
# Usage:
#   make seed                         # Runs regular-12 scenario
#   make seed SCENARIO=irregular      # Runs a different scenario
#   make seed CYCLES=20               # Overrides cycle count
#   make seed SEED=12345              # Sets PRNG seed for reproducibility
seed:
	go run ./engine/cmd/seed/ --scenario=$(SCENARIO) --db=openmenses.db --cycles=$(CYCLES) --seed=$(SEED)

# Generate all built-in scenarios into separate database files for UI testing.
# Creates files: openmenses-regular-12.db, openmenses-irregular.db, etc.
# Usage:
#   make seed-all                     # Runs all scenarios with defaults
seed-all:
	go run ./engine/cmd/seed/ --scenario=regular-12 --db=openmenses-regular-12.db
	go run ./engine/cmd/seed/ --scenario=ovulatory-somewhat-irregular --db=openmenses-ovulatory-somewhat-irregular.db
	go run ./engine/cmd/seed/ --scenario=ovulatory-very-irregular --db=openmenses-ovulatory-very-irregular.db
	go run ./engine/cmd/seed/ --scenario=ovulatory-unknown --db=openmenses-ovulatory-unknown.db
	go run ./engine/cmd/seed/ --scenario=hormonal-regular --db=openmenses-hormonal-regular.db
	go run ./engine/cmd/seed/ --scenario=hormonal-somewhat-irregular --db=openmenses-hormonal-somewhat-irregular.db
	go run ./engine/cmd/seed/ --scenario=hormonal-very-irregular --db=openmenses-hormonal-very-irregular.db
	go run ./engine/cmd/seed/ --scenario=irregular --db=openmenses-irregular.db
	go run ./engine/cmd/seed/ --scenario=irregular-very-irregular --db=openmenses-irregular-very-irregular.db
	go run ./engine/cmd/seed/ --scenario=shortening --db=openmenses-shortening.db
	go run ./engine/cmd/seed/ --scenario=medication-gaps --db=openmenses-medication-gaps.db
	go run ./engine/cmd/seed/ --scenario=minimal --db=openmenses-minimal.db

# Generate persona fixture JSON files for the UI Dev Tools.
# Output: ui/public/fixtures/<scenario-name>.json
# These are pre-generated export files that the UI imports client-side.
fixtures-generate:
	mkdir -p ui/public/fixtures
	go run ./engine/cmd/seed/ --scenario=regular-12 --db=:memory: --export=ui/public/fixtures/regular-12.json
	go run ./engine/cmd/seed/ --scenario=ovulatory-somewhat-irregular --db=:memory: --export=ui/public/fixtures/ovulatory-somewhat-irregular.json
	go run ./engine/cmd/seed/ --scenario=ovulatory-very-irregular --db=:memory: --export=ui/public/fixtures/ovulatory-very-irregular.json
	go run ./engine/cmd/seed/ --scenario=ovulatory-unknown --db=:memory: --export=ui/public/fixtures/ovulatory-unknown.json
	go run ./engine/cmd/seed/ --scenario=hormonal-regular --db=:memory: --export=ui/public/fixtures/hormonal-regular.json
	go run ./engine/cmd/seed/ --scenario=hormonal-somewhat-irregular --db=:memory: --export=ui/public/fixtures/hormonal-somewhat-irregular.json
	go run ./engine/cmd/seed/ --scenario=hormonal-very-irregular --db=:memory: --export=ui/public/fixtures/hormonal-very-irregular.json
	go run ./engine/cmd/seed/ --scenario=irregular --db=:memory: --export=ui/public/fixtures/irregular.json
	go run ./engine/cmd/seed/ --scenario=irregular-very-irregular --db=:memory: --export=ui/public/fixtures/irregular-very-irregular.json

# ---------------------------------------------------------------------------
# Mobile targets (iOS/Android native shells)
# ---------------------------------------------------------------------------

# Install gomobile tooling (one-time setup).
mobile-setup:
	go install golang.org/x/mobile/cmd/gomobile@latest
	go install golang.org/x/mobile/cmd/gobind@latest
	PATH="$(shell go env GOPATH)/bin:$$PATH" $(shell go env GOPATH)/bin/gomobile init

# Build UI production bundle into ui/dist/.
ui-bundle: ui-build

# Build iOS framework via gomobile bind.
# Requires: Xcode CLI tools, gomobile (run `make mobile-setup` first).
# Output: mobile/ios/Engine.xcframework/
# Note: This target only works on macOS with Xcode installed.
mobile-ios:
	PATH="$(shell go env GOPATH)/bin:$$PATH" $(shell go env GOPATH)/bin/gomobile bind -target=ios -o mobile/ios/Engine.xcframework ./engine/mobile/

# Regenerate the Xcode project from project.yml via xcodegen.
# Run this whenever a Swift source file is added or removed, or project.yml
# changes. Xcode cannot self-heal — the .pbxproj must be updated before
# building, because script phases run too late to affect the Compile Sources list.
# Requires: xcodegen (brew install xcodegen).
mobile-project:
	cd mobile/ios && xcodegen generate

# ---------------------------------------------------------------------------
# Aggregate targets
# ---------------------------------------------------------------------------

# Run all linters
lint: proto-lint engine-lint ui-lint

# Run all tests
test: engine-test ui-test

# Full CI validation (linting + generation check + tests)
ci: proto-lint proto-check engine-lint engine-test ui-lint ui-test
