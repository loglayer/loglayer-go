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
build: ## Build every module (mirrors CI's build step across all sub-modules).
	scripts/foreach-module.sh build

.PHONY: test
test: ## Fast main-module tests (no race, no sub-modules). Inner-loop only.
	$(GO) test ./...

.PHONY: test-race
test-race: ## Race tests across every module, parallelized. Mirrors pre-push.
	scripts/foreach-module.sh test

.PHONY: bench
bench: ## Run all benchmarks (no tests).
	$(GO) test -bench=. -benchmem -run=^$$ -benchtime=1s .

##@ Lint & Format

.PHONY: vet
vet: ## go vet across every module.
	scripts/foreach-module.sh vet

.PHONY: fmt
fmt: ## Format every Go file in place with gofmt.
	$(GOFMT) -w $(GO_FILES)

.PHONY: fmt-check
fmt-check: ## Fail if any module has files that aren't gofmt-clean.
	scripts/foreach-module.sh fmt

.PHONY: tidy
tidy: ## go mod tidy across every module + diff check (fails on drift).
	scripts/foreach-module.sh tidy

.PHONY: staticcheck
staticcheck: ## staticcheck across every shipped module. Same set CI runs.
	scripts/foreach-module.sh staticcheck

.PHONY: vuln
vuln: ## govulncheck across every shipped module.
	scripts/foreach-module.sh vuln

.PHONY: lint
lint: vet fmt-check staticcheck ## vet + fmt-check + staticcheck across every module.

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

.PHONY: toc
toc: ## Regenerate the README table of contents (auto-run by pre-commit).
	$(BUN) run toc

##@ Live integration tests

.PHONY: livetest-datadog
livetest-datadog: ## Send a real log to Datadog. Requires DD_API_KEY.
	$(GO) test -tags=livetest -v -run TestLive_Datadog ./transports/datadog/

##@ CI

.PHONY: ci
ci: tidy lint test-race ## Full CI gauntlet: tidy + lint + multi-module race tests. Mirror locally before pushing.
