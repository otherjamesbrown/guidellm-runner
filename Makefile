.PHONY: build run test clean tidy fmt lint

BINARY_NAME=guidellm-runner
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/runner

run: build
	$(BUILD_DIR)/$(BINARY_NAME) -config configs/config.yaml

run-debug: build
	$(BUILD_DIR)/$(BINARY_NAME) -config configs/config.yaml -log-level debug

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
	go clean

tidy:
	go mod tidy

fmt:
	go fmt ./...

lint:
	golangci-lint run

# Development helpers
dev-deps:
	pip install "guidellm[recommended]"

# Docker targets
docker-build:
	docker build -t guidellm-runner:latest .

docker-run:
	docker run -p 9090:9090 -v $(PWD)/configs:/app/configs guidellm-runner:latest
