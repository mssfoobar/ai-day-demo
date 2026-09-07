### Variables
LOCALBIN := .bin

# Log Colors
COLOR := "\e[1;36m%s\e[0m\n"
RED :=   "\e[1;31m%s\e[0m\n"

### Install Tools

# Create local bin directory
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

# Install golangci-lint
GOLANGCI_LINT_VERSION := {GOLANGCI_LINT_VERSION}
GOLANGCI_LINT := $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# Install swag
SWAG_VERSION := {SWAG_VERSION}
SWAG := $(LOCALBIN)/swag-$(SWAG_VERSION)
$(SWAG): $(LOCALBIN)
	$(call go-install-tool,$(SWAG),github.com/swaggo/swag/cmd/swag,$(SWAG_VERSION))

# Install mockery
MOCK_VERSION := {MOCKERY_VERSION}
MOCK := $(LOCALBIN)/mockery-$(MOCK_VERSION)
$(MOCK): $(LOCALBIN)
	$(call go-install-tool,$(MOCK),github.com/vektra/mockery/v3,$(MOCK_VERSION))

# Install golang-migrate
MIGRATE_VERSION := {MIGRATE_VERSION}
MIGRATE := $(LOCALBIN)/migrate-$(MIGRATE_VERSION)
$(MIGRATE): $(LOCALBIN)
	$(call go-install-tool,$(MIGRATE),github.com/golang-migrate/migrate/v4/cmd/migrate,$(MIGRATE_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
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

### Swagger
.PHONY: swag
swag: $(SWAG)
	@printf $(COLOR) "Generating swagger docs..."
	@$(SWAG) init -d ./cmd/server,./internal/handler -o ./docs --parseDependency --parseInternal

### Mock generation
.PHONY: mock
mock: $(MOCK)
	@printf $(COLOR) "Generating mocks..."
	@$(MOCK)

### Formatting — applies the enabled formatters (gci, gofmt) in-place.
### Run before lint-code; the code generators don't pre-format output.
.PHONY: fmt
fmt: $(GOLANGCI_LINT)
	@printf $(COLOR) "Formatting code..."
	@$(GOLANGCI_LINT) fmt

### Checks
.PHONY: lint-code
lint-code: $(GOLANGCI_LINT)
	@printf $(COLOR) "Linting code..."
	@$(GOLANGCI_LINT) run --verbose

### Test — regenerates mocks first so a stale .mockery.yaml never masks
### a real "undefined: mocks.NewFoo" failure.
.PHONY: test
test: mock
	@printf $(COLOR) "Running tests..."
	@go test ./...

### Build & Run
SERVER := $(LOCALBIN)/server
run: build
	@$(SERVER) --log.level=debug

build: $(LOCALBIN)
	@go build -o $(SERVER) ./cmd/server

### Migrations — golang-migrate CLI for generating numbered .up/.down SQL files.
### Service runs migration up on startup via repo.RunMigrations().
### Usage: make migrate-create NAME=incident  (creates schema/0001_incident.{up,down}.sql)
.PHONY: migrate-create
migrate-create: $(MIGRATE)
	@if [ -z "$(NAME)" ]; then \
		printf $(RED) "NAME is required, e.g. make migrate-create NAME=incident"; \
		exit 1; \
	fi
	@mkdir -p schema
	@printf $(COLOR) "Generating migration schema/$(NAME)..."
	@$(MIGRATE) create -ext sql -seq -digits 4 -dir schema $(NAME)
