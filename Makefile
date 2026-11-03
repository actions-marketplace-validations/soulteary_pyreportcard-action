.PHONY: build test integration
build:
	go build -trimpath -o dist/pyreportcard-cli ./cmd/pyreportcard-cli
test:
	go test -race ./...
	go vet ./...
	bash -n scripts/*.sh
	./scripts/test-prepare-release.sh
integration:
	./scripts/integration.sh
