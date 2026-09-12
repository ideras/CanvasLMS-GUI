# Convenience wrappers around the Wails v3 task runner.
# Requires wails3 on PATH (pinned: github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.20).
WAILS3 = $(shell command -v wails3 2>/dev/null || echo $(shell go env GOPATH)/bin/wails3)

.PHONY: build dev clean

build:
	$(WAILS3) task build

dev:
	$(WAILS3) task dev

clean:
	rm -rf bin build/bin
	$(WAILS3) task common:install:frontend:deps --force 2>/dev/null || true
	rm -rf frontend/dist frontend/node_modules frontend/bindings