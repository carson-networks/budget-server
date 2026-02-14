GOCMD=go
BINARY_NAME=weather-server

GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all test build vendor

all: help

## Build:
build: ## Build your project and put the output binary in out/bin/
	mkdir -p out/bin
	GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME) .

clean: ## Remove build related file
	rm -fr ./bin
	rm -fr ./out

vendor: ## Copy of all packages needed to support builds and tests in the vendor directory
	$(GOCMD) mod vendor

serve: ## Run the go code
	docker compose --profile serve up

serve-local: ## Run the go code changes that are locally made
	docker image rm cdmeyer/budget-server:local || true
	docker compose --profile serve-local build
	docker compose --profile serve-local up

serve-local-rebuild: ## Rebuild the local changes
	docker compose up --detach --build budget-server-local

rebuild-serve:
	GO111MODULE=on GOARCH=arm64 GOOS=linux $(GOCMD) build -o out/bin/$(BINARY_NAME) .
	docker stop server-local
	docker cp out/bin/$(BINARY_NAME) server-local:/budget-server
	docker start server-local

## Help:
help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

format: ## Format imports
	find . -type f -name '*.go' -not -path "./vendor/*" | xargs goimports -local -l -w

generate-mocks: ## Generate mocks
	mockery

setup:
	# Install goimports
	go install golang.org/x/tools/cmd/goimports@v0.31.0
	# Install mockery
	go install github.com/vektra/mockery/v2@v2.53.3
