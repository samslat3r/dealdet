.PHONY: build build-api build-worker clean test help

help:
@echo "Available targets:"
@echo "  build          - Build all binaries (api and worker)"
@echo "  build-api      - Build API binary"
@echo "  build-worker   - Build worker binary"
@echo "  clean          - Remove build artifacts"
@echo "  test           - Run tests"
@echo "  help           - Show this help message"

build: build-api build-worker

build-api:
go build -o bin/api ./cmd/api

build-worker:
go build -o bin/worker ./cmd/worker

clean:
rm -rf bin/

test:
go test ./...
