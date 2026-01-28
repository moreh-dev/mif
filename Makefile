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
	@helm lint ./deploy/helm/*

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

.PHONY: test-e2e
test-e2e: ## Run all E2E tests (performance + quality).
	@$(MAKE) test-e2e-performance
	@$(MAKE) test-e2e-quality

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

.PHONY: test-e2e-clean
test-e2e-clean: ## Manually clean up E2E test resources.
	@echo "Deleting kind cluster 'mif-e2e' (if it exists)..."
	@kind get clusters | grep -q '^mif-e2e$$' && kind delete cluster --name mif-e2e || echo "kind cluster 'mif-e2e' not found, nothing to delete."

.PHONY: test-e2e-env
test-e2e-env: ## Display E2E test environment variables (auto-generated from code).
	@go run -tags=e2e ./test/cmd/printenv env
