# Include .env file if it exists
-include .env
export
export CGO_CFLAGS=-isysroot /Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk
export CGO_LDFLAGS=-isysroot /Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk

# Package path for ldflags
PACKAGE_PATH=github.com/focusd-so/focusd/internal/identity

# Default values for hex parts - replace these or pass them via command line
# These will use values from .env if defined there
HEX_P1 ?= ""
HEX_P2 ?= ""
HEX_P3 ?= ""

LDFLAGS=-X '$(PACKAGE_PATH).CompileTimeHexP1=$(HEX_P1)' \
        -X '$(PACKAGE_PATH).CompileTimeHexP2=$(HEX_P2)' \
        -X '$(PACKAGE_PATH).CompileTimeHexP3=$(HEX_P3)'

.PHONY: build
build:
	GOFLAGS="-tags=productio" wails3 build LDFLAGS="$(LDFLAGS)"

.PHONY: cli
cli:
	go build -ldflags "$(LDFLAGS)" -o bin/focusd-cli cmd/main.go
	
.PHONY: dev
dev:
	wails3 dev --port 17000 LDFLAGS="$(LDFLAGS)"

.PHONY: tidy
tidy:
	go mod tidy
