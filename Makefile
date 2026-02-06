.PHONY: install_lint lint install_openapi swagger install_buf gen_proto

install_lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.6

lint:
	golangci-lint run ./...

# After install source PATH files
install_openapi:
	go install github.com/swaggo/swag/cmd/swag@latest

swagger:
	swag init -g cmd/main.go --parseDependency --parseInternal \
	-o ./docs \
	--outputTypes yaml,json \
	-q
	swag fmt

