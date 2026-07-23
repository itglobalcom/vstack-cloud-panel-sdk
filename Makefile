-include .env
export
BINARY_NAME=vstack-cloud-panel-sdk

.PHONY: test clean examples

test:
	@go test -v ./...

clean:
	@go clean && rm -rf bin/

deps:
	@go mod download && go mod tidy

fmt:
	@go fmt ./...

vet:
	@go vet ./...

example: 
	@go run ./examples -resource $(RESOURCE) -api-key $(API_KEY) -api-url $(API_URL)