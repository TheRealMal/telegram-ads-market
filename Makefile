.PHONY: install_lint lint install_openapi swagger install_buf gen_proto

install_lint: ## Install linting tool
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.6

lint: ## Run linting
	golangci-lint run ./...

install_openapi: ## Install swagger generator package
	go install github.com/swaggo/swag/cmd/swag@latest

swagger: ## Generate swagger documentation
	swag init -g cmd/main.go --parseDependency --parseInternal \
	-o ./docs \
	--outputTypes yaml,json \
	-q
	swag fmt


help: ## Show this help message
	@echo "Platform Scripts"
	@echo ""
	@echo "Usage: make <target> [ARGS=\"...\"]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""