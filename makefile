BIN_DIR?=$(CURDIR)/.bin
JB_BIN=$(BIN_DIR)/jb
JSONNET_BIN=$(BIN_DIR)/jsonnet
JSONNET_FMT_BIN=$(BIN_DIR)/jsonnetfmt
GRAFANA_DASHBOARD_LINTER_BIN=$(BIN_DIR)/dashboard-linter
JSONNET_LINT_BIN=$(BIN_DIR)/jsonnet-lint
TOOLING=$(JB_BIN) $(JSONNET_BIN) $(JSONNET_FMT_BIN) $(JSONNET_LINT_BIN) $(GRAFANA_DASHBOARD_LINTER_BIN)
JSONNET_VENDOR="misc/dashboards/vendor"
GO_SRCS := $(shell find . -name '*.go' -not -path "./vendor/*")
JSONNET_FILES := $(shell find misc/dashboards -name 'vendor' -prune -o -name '*.libsonnet' -print -o -name '*.jsonnet' -print)

.PHONY: all
all: dashboards pbs_exporter

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(TOOLING): $(BIN_DIR)
	@echo "Installing tools from hack/tools.go"
	@cd hack && go list \
		-e -mod=mod -tags tools -f '{{ range .Imports }}{{ printf "%s\n" .}}{{end}}' ./ | \
		xargs -tI % go build -mod=mod -o $(BIN_DIR) %

$(JSONNET_VENDOR): $(JB_BIN) misc/dashboards/jsonnetfile.json
	@cd misc/dashboards && $(JB_BIN) install

.PHONY: test
test: $(BIN_DIR) $(JSONNET_VENDOR)
	@echo "Running tests..."
	@go test -v -race -shuffle=on -coverprofile=coverage.out ./...

.PHONY: coverage
coverage: test
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out

.PHONY: dashboards
dashboards: $(JSONNET_VENDOR) misc/dashboards/dashboards.jsonnet
	@echo "Generating dashboards..."
	@$(JSONNET_BIN) --jpath $(JSONNET_VENDOR) --create-output-dirs --multi misc/dashboards/build misc/dashboards/dashboards.jsonnet

.PHONY: fmt jsonnet-fmt go-fmt
fmt: jsonnet-fmt go-fmt

jsonnet-fmt: $(JSONNET_FMT_BIN)
	@echo "Formatting Jsonnet files..."
	@$(JSONNET_FMT_BIN) -i $(JSONNET_FILES)

go-fmt:
	@echo "Formatting Go files..."
	@gofmt -s -w .

go-vet:
	@echo "Running Go vet..."
	@go vet ./...

.PHONY: lint dashboards-lint jsonnet-lint
lint: $(BIN_DIR) $(JSONNET_VENDOR) dashboards-lint jsonnet-lint go-vet

dashboards-lint: $(GRAFANA_DASHBOARD_LINTER_BIN) dashboards
	@echo "Linting Grafana dashboards..."
	@find misc/dashboards/build -name '*.json' -print0 | \
		xargs -n 1 -0 $(GRAFANA_DASHBOARD_LINTER_BIN) lint -c .lint --strict

jsonnet-lint: $(JSONNET_FILES:%=%.jsonnet-lint)
	@echo "All Jsonnet files linted."

%.jsonnet-lint:
	$(JSONNET_LINT_BIN) -J misc/dashboards/vendor -- $*

pbs_exporter: $(GO_SRCS)
	@echo "Building pbs_exporter..."
	@go get ./cmd/pbs_exporter
	@go build -v -o pbs_exporter ./cmd/pbs_exporter

.PHONY: clean
clean:
	rm -rf $(JSONNET_VENDOR) misc/dashboards/build $(BIN_DIR) coverage.out
