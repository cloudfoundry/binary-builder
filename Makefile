.PHONY: test unit-test unit-test-race parity-test parity-test-all exerciser-test

# Tier 1: unit tests (no Docker, no network)
unit-test:
	go test ./...

# Tier 1 with race detector
unit-test-race:
	go test -race ./...

# Tier 2: parity test for a single dep (requires Docker + network)
# Usage: make parity-test DEP=ruby VERSION=3.3.6 SHA256=abc123 STACK=cflinuxfs4
STACK ?= cflinuxfs4
parity-test:
	@test -n "$(DEP)"     || (echo "DEP is required"; exit 1)
	@test -n "$(VERSION)" || (echo "VERSION is required"; exit 1)
	@test -n "$(SHA256)"  || (echo "SHA256 is required"; exit 1)
	./test/parity/compare-builds.sh "$(DEP)" "$(VERSION)" "$(SHA256)" "$(STACK)"

# Tier 2: parity test for all deps in the matrix (requires Docker + network)
parity-test-all:
	./test/parity/run-all.sh "$(STACK)"

# Tier 3: exerciser test for a single artifact (requires Docker)
# Usage: make exerciser-test ARTIFACT=/tmp/ruby_3.3.6_...tgz STACK=cflinuxfs4
exerciser-test:
	@test -n "$(ARTIFACT)" || (echo "ARTIFACT is required"; exit 1)
	@test -n "$(STACK)"    || (echo "STACK is required"; exit 1)
	ARTIFACT="$(ARTIFACT)" STACK="$(STACK)" \
	  go test -tags integration ./test/exerciser/ -v

# Run Tier 1 + Tier 2 (requires Docker + network)
test: unit-test parity-test-all
