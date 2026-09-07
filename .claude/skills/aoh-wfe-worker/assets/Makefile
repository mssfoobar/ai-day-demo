### Variables
LOCALBIN := .bin

COLOR := "\e[1;36m%s\e[0m\n"

# Create local bin directory
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

# Install golangci-lint
GOLANGCI_LINT_VERSION := v2.10.1
GOLANGCI_LINT := $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
printf $(COLOR) "Downloading $${package}" ;\
tmpdir=$$(mktemp -d) ;\
GOBIN=$${tmpdir} go install $${package} ;\
mv $${tmpdir}/$$(basename "$$(echo "$(1)" | sed "s/-$(3)$$//")") $(1) ;\
rm -rf $${tmpdir} ;\
}
endef

WORKER := $(LOCALBIN)/worker

.PHONY: build
build: $(LOCALBIN)
	@go build -o $(WORKER) ./cmd/worker

.PHONY: run
run: build
	@$(WORKER) --log.level=debug

.PHONY: test
test:
	@printf $(COLOR) "Running tests..."
	@go test ./...

.PHONY: fmt
fmt: $(GOLANGCI_LINT)
	@printf $(COLOR) "Formatting code..."
	@$(GOLANGCI_LINT) fmt

.PHONY: lint-code
lint-code: $(GOLANGCI_LINT)
	@printf $(COLOR) "Linting code..."
	@$(GOLANGCI_LINT) run --verbose
