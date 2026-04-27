# LogLayer for Go — task runner.
#
# Run `make` (or `make help`) to list targets.
# Run `make <target>` to invoke one.
#
# Tip: tab characters are required for recipe lines (Make's syntax). If a
# target stops working after editing, the most likely cause is spaces.

GO        ?= go
GOFMT     ?= gofmt
LEFTHOOK  ?= lefthook
BUN       ?= bun

# Module-relative source list; excludes vendored code and node_modules.
GO_FILES  := $(shell find . -name '*.go' -not -path './docs/node_modules/*')

.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Print this help.
	@awk 'BEGIN { FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n" } \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' \
		$(MAKEFILE_LIST)

##@ Build & Test

.PHONY: build
build: ## Compile every package (sanity check; library has no binary).
	$(GO) build ./...

.PHONY: test
test: ## Run all tests.
	$(GO) test ./...

.PHONY: test-race
test-race: ## Run all tests with the race detector.
	$(GO) test -race -timeout 60s ./...

.PHONY: bench
bench: ## Run all benchmarks (no tests).
	$(GO) test -bench=. -benchmem -run=^$$ -benchtime=1s .

##@ Lint & Format

.PHONY: vet
vet: ## go vet over every package.
	$(GO) vet ./...

.PHONY: fmt
fmt: ## Format every Go file in place with gofmt.
	$(GOFMT) -w $(GO_FILES)

.PHONY: fmt-check
fmt-check: ## Fail if any Go file is not gofmt-clean.
	@unformatted="$$($(GOFMT) -l $(GO_FILES))"; \
	if [ -n "$$unformatted" ]; then \
		echo "These files are not gofmt-clean:"; \
		echo "$$unformatted"; \
		echo; \
		echo "Run: make fmt"; \
		exit 1; \
	fi

.PHONY: lint
lint: vet fmt-check ## vet + fmt-check.

##@ Docs

.PHONY: docs
docs: ## Build the VitePress docs site (output in docs/.vitepress/dist).
	cd docs && $(BUN) run docs:build

.PHONY: docs-dev
docs-dev: ## Run the VitePress dev server with live reload.
	cd docs && $(BUN) run docs:dev

##@ Tooling

.PHONY: hooks
hooks: ## Install git pre-commit/pre-push hooks via lefthook.
	$(LEFTHOOK) install

##@ Live integration tests

.PHONY: livetest-datadog
livetest-datadog: ## Send a real log to Datadog. Requires DD_API_KEY.
	$(GO) test -tags=livetest -v -run TestLive_Datadog ./transports/datadog/

##@ CI

.PHONY: ci
ci: lint test-race ## What CI runs: lint + test-race. Mirror locally before pushing.
