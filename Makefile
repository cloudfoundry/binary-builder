.PHONY: test unit-test unit-test-race parity-test parity-test-all exerciser-test

# Tier 1: unit tests (no Docker, no network)
unit-test:
	go test ./...

# Tier 1 with race detector
unit-test-race:
	go test -race ./...

# Tier 2: parity test for a single dep (requires Docker + network)
# Uses the same data.json values as parity-test-all (defined in run-all.sh).
# Usage: make parity-test DEP=httpd [STACK=cflinuxfs4]
STACK ?= cflinuxfs4
parity-test:
	@test -n "$(DEP)" || (echo "DEP is required. Usage: make parity-test DEP=<name> [STACK=<stack>]"; exit 1)
	DEP=$(DEP) ./test/parity/run-all.sh "$(STACK)"

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
