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
	
server:
	go run cmd/main.go serve 

wails3:
	wails3 dev --port 17000 LDFLAGS="$(LDFLAGS)"

.PHONY: dev
dev: server wails3 &> /dev/null &

.PHONY: tidy
tidy:
	go mod tidy

# Deployment
SERVICE_NAME ?= api
REGION ?= us-east4
IMAGE_TAG ?= $(shell git rev-parse HEAD)
IMAGE_URL ?= $(REGION)-docker.pkg.dev/focusd-466610/cloud-run-source-deploy/focusd/api:$(IMAGE_TAG)

.PHONY: deploy-api
deploy-api:
	gcloud run deploy $(SERVICE_NAME) \
		--image=$(IMAGE_URL) \
		--region=$(REGION) \
		--port=8080 \
		--use-http2 \
		--cpu=1 \
		--memory=512Mi \
		--concurrency=80 \
		--timeout=300 \
		--max-instances=20 \
		--cpu-boost \
		--service-account=985133598369-compute@developer.gserviceaccount.com \
		--ingress=all \
		--max-request-body-size=20Ki \
		--set-env-vars="TURSO_CONNECTION_PATH=libsql://focusd-aram.aws-us-east-1.turso.io,POLAR_SERVER=production" \
		--set-secrets="TURSO_CONNECTION_TOKEN=turso_connection_token:latest,PASETO_KEYS=paseto_keys:latest,HMAC_SECRET_KEY=hmac_secret_key:latest,GEMINI_API_KEY=gemini_api_key:2,POLAR_WEBHOOK_SECRET=POLAR_WEBHOOK_SECRET:latest,POLAR_ACCESS_TOKEN=polar_access_token:latest"
