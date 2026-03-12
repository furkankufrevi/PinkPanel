VERSION ?= $(shell git describe --tags 2>/dev/null || echo "0.3.3139-alpha")
LDFLAGS = -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: dev build test lint fmt clean

## dev: Run air (Go hot-reload) + vite dev concurrently
dev:
	@echo "Starting PinkPanel development servers..."
	@trap 'kill 0' EXIT; \
		cd web && npm run dev & \
		air & \
		wait

## build: Build frontend + compile Go binaries
build: build-frontend build-backend

build-frontend:
	cd web && npm ci && npm run build
	@rm -rf cmd/server/static
	@cp -r web/dist cmd/server/static

build-backend:
	CGO_ENABLED=0 go build $(LDFLAGS) -o dist/pinkpanel ./cmd/server
	CGO_ENABLED=0 go build $(LDFLAGS) -o dist/pinkpanel-agent ./cmd/agent
	CGO_ENABLED=0 go build $(LDFLAGS) -o dist/pinkpanel-cli ./cmd/cli

## test: Run all Go tests
test:
	go test ./...

## lint: Run golangci-lint + eslint
lint: lint-go lint-frontend

lint-go:
	golangci-lint run ./...

lint-frontend:
	cd web && npx eslint src/

## fmt: Run gofmt + prettier
fmt:
	gofmt -w .
	cd web && npx prettier --write "src/**/*.{ts,tsx,css}"

## clean: Remove build artifacts
clean:
	rm -rf dist/ tmp/ cmd/server/static/
	cd web && rm -rf dist/ node_modules/.vite/
