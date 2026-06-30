GO ?= go

INPUT  ?= testdata/example.csv
OUTPUT ?= testdata/example.csv
MIN    ?= 3
MAX    ?= 4

.PHONY: help build run test fmt vet clean docker-build docker-run

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

build: ## Build the binary to bin/teaming
	$(GO) build -o bin/teaming ./cmd/teaming

run: ## Run without building (pass extra flags via ARGS=)
	$(GO) run ./cmd/teaming $(ARGS)

test: ## Run all tests
	$(GO) test ./...

fmt: ## Format all Go files with gofumpt
	$(GO) tool gofumpt -w .

vet: ## Run go vet
	$(GO) vet ./...

clean: ## Remove the bin/ directory
	rm -rf bin/

docker-build: ## Build the Docker image (teaming:latest)
	docker build -f cmd/teaming/Dockerfile -t teaming:latest .

docker-run: ## Run the image against a mounted CSV (INPUT, OUTPUT, MIN, MAX)
	docker run --rm -v "$(PWD)":/data teaming:latest \
		-i /data/$(INPUT) \
		-o /data/$(OUTPUT) \
		--min $(MIN) \
		--max $(MAX)
