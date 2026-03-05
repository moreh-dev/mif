##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: helm-lint
helm-lint: ## Lint Helm charts.
	@for chart in ./deploy/helm/*; do \
	  if [ -d "$$chart" ] && [ -f "$$chart/Chart.yaml" ]; then \
	    helm lint "$$chart"; \
	  fi; \
	done

.PHONY: helm-docs
helm-docs: ## Generate Helm chart documentation.
	@helm-docs --chart-search-root=./deploy/helm

.PHONY: helm-dependency
helm-dependency: ## Update Helm chart dependencies.
	@for chart in ./deploy/helm/*; do \
	  if [ -d "$$chart" ] && [ -f "$$chart/Chart.yaml" ]; then \
	    echo "Updating dependencies for $$chart"; \
	    helm dependency update "$$chart"; \
	  fi; \
	done

##@ Testing

KIND_CLUSTER ?= mif-e2e

.PHONY: test-e2e
test-e2e: ## Run all E2E tests (smoke + performance + quality).
	@$(MAKE) test-e2e-smoke
	@$(MAKE) test-e2e-performance
	@$(MAKE) test-e2e-quality

.PHONY: test-e2e-smoke
test-e2e-smoke: ## Run smoke E2E tests.
	@mkdir -p test-reports
	@go test -tags=e2e -v ./test/e2e/smoke/... -timeout 30m \
		-ginkgo.v \
		-ginkgo.label-filter=smoke \
		-ginkgo.junit-report="$(CURDIR)/test-reports/junit-smoke.xml" \
		-ginkgo.json-report="$(CURDIR)/test-reports/report-smoke.json"

.PHONY: test-e2e-performance
test-e2e-performance: ## Run inference-perf performance tests.
	@mkdir -p test-reports
	@go test -tags=e2e -v ./test/e2e/performance/... -timeout 30m \
		-ginkgo.v \
		-ginkgo.label-filter=performance \
		-ginkgo.junit-report="$(CURDIR)/test-reports/junit-performance.xml" \
		-ginkgo.json-report="$(CURDIR)/test-reports/report-performance.json"

.PHONY: test-e2e-quality
test-e2e-quality: ## Run quality benchmark tests.
	@mkdir -p test-reports
	@go test -tags=e2e -v ./test/e2e/quality/... -timeout 30m \
		-ginkgo.v \
		-ginkgo.label-filter=quality \
		-ginkgo.junit-report="$(CURDIR)/test-reports/junit-quality.xml" \
		-ginkgo.json-report="$(CURDIR)/test-reports/report-quality.json"

.PHONY: setup-test-e2e
setup-test-e2e: ## Create Kind cluster for e2e tests (idempotent).
	@command -v kind >/dev/null 2>&1 || { echo "kind is not installed."; exit 1; }
	@case "$$(kind get clusters)" in \
		*"$(KIND_CLUSTER)"*) echo "Kind cluster '$(KIND_CLUSTER)' already exists." ;; \
		*) echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; kind create cluster --name $(KIND_CLUSTER) ;; \
	esac

.PHONY: cleanup-test-e2e
cleanup-test-e2e: ## Delete Kind cluster used for e2e tests.
	@kind delete cluster --name $(KIND_CLUSTER) 2>/dev/null || true

.PHONY: test-e2e-kind
test-e2e-kind: setup-test-e2e ## Run smoke e2e tests on a local Kind cluster.
	@SKIP_PREREQUISITE=false $(MAKE) test-e2e-smoke
	@$(MAKE) cleanup-test-e2e

.PHONY: test-e2e-env
test-e2e-env: ## Display E2E test environment variables (auto-generated from code).
	@go run -tags=e2e ./test/cmd/printenv env
