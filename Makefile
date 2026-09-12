WAILS = $(shell go env GOPATH)/bin/wails
TAGS  = webkit2_41

.PHONY: build dev clean

build:
	$(WAILS) build -tags $(TAGS)

dev:
	$(WAILS) dev -tags $(TAGS)

clean:
	rm -rf build/bin
