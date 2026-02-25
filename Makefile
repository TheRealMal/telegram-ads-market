.PHONY: install_lint lint install_openapi swagger install_buf gen_proto build_linux_bin docker_build_bin

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

init_userbot: ## Initialize the userbot session file
	docker compose up -d postgres redis migrations
	docker compose run --rm -it userbot

init_certs: ## Initialize the certificates
	docker compose run --rm certbot

start: ## Start the server
	docker compose up -d

rollout_web: ## Rebuild web and deploy updated containers
	docker compose build web
	docker compose up web -d --force-recreate

rollout_back: ## Rebuild backend and deploy updated containers
	docker compose build api
	docker compose up api bot userbot blockchain_observer -d --force-recreate

stop: ## Stop the server
	docker compose down

docker_build_bin: ## Build binary via Docker and save to ./build/server (requires BuildKit)
	DOCKER_BUILDKIT=1 docker build --platform linux/amd64 --output type=local,dest=./build/bin/ --target=artifact -f Dockerfile .

help: ## Show this help message
	@echo "Platform Scripts"
	@echo ""
	@echo "Usage: make <target> [ARGS=\"...\"]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""