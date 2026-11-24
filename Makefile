SHELL := /bin/sh

APP := cloudforge
IMAGE ?= ghcr.io/OWNER/cloudforge
KIND_CLUSTER ?= cloudforge
NAMESPACE ?= cloudforge

.PHONY: all build test lint fmt vet tidy install kind-up kind-down kind-load helm-lint helm-template examples api run clean

all: fmt vet test build

build:
	mkdir -p bin
	go build -trimpath -o bin/$(APP) ./cmd/cloudforge
	go build -trimpath -o bin/cloudforge-api ./cmd/api

test:
	go test ./...

lint:
	command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint is required" && exit 1)
	golangci-lint run ./...

fmt:
	gofmt -w cmd internal

vet:
	go vet ./...

tidy:
	go mod tidy

install:
	go install ./cmd/cloudforge

kind-up:
	KIND_CLUSTER=$(KIND_CLUSTER) ./scripts/kind-up.sh

kind-down:
	kind delete cluster --name $(KIND_CLUSTER)

kind-load:
	docker build -t cloudforge/node-app:local examples/node-app
	kind load docker-image cloudforge/node-app:local --name $(KIND_CLUSTER)

helm-lint:
	helm lint charts/cloudforge-app

helm-template:
	helm template demo charts/cloudforge-app --namespace $(NAMESPACE) --set image.repository=$(IMAGE) --set image.tag=local

examples: helm-lint helm-template

api: build
	./bin/cloudforge-api --listen :8080

run: build
	./bin/cloudforge --help

clean:
	rm -rf bin coverage.out
