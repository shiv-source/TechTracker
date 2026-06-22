.PHONY: build run test lint vet clean clean-stack clean-bin docker all ui-dev ui-build ui-install

APP_NAME := techtracker
BIN_DIR := bin
BINARY := $(BIN_DIR)/main

# ── Go (1.26) ──────────────────────────────────────────────

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) ./cmd/$(APP_NAME)

run: build
	$(BINARY)

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)

clean-bin:
	rm -rf $(BIN_DIR)

clean-stack:
	rm -rf data readme.md

docker:
	docker build -t $(APP_NAME) .

all: clean-stack build run

# ── UI (Node.js 24 · pnpm v11 workspace) ───────────────────

ui-install:
	pnpm install

ui-dev:
	pnpm --filter techtracker-ui dev

ui-build:
	pnpm --filter techtracker-ui build

ui-clean:
	rm -rf ui/dist ui/node_modules
