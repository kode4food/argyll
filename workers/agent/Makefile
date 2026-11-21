GO ?= go

.PHONY: format check test pre-commit refresh-openai-models refresh-anthropic-models refresh-bedrock-models refresh-gemini-models

format:
	$(GO) tool goimports -w .
	$(GO) fix ./...

check:
	$(GO) vet ./...
	$(GO) tool staticcheck ./...

test:
	GOWORK=off $(GO) test -mod=readonly ./...

pre-commit: format check test

refresh-openai-models:
	GOWORK=off $(GO) run -mod=readonly ./cmd/argyll-agent-openai/internal/refresh

refresh-anthropic-models:
	GOWORK=off $(GO) run -mod=readonly ./cmd/argyll-agent-anthropic/internal/refresh

refresh-bedrock-models:
	GOWORK=off $(GO) run -mod=readonly ./cmd/argyll-agent-bedrock/internal/refresh

refresh-gemini-models:
	GOWORK=off $(GO) run -mod=readonly ./cmd/argyll-agent-gemini/internal/refresh
