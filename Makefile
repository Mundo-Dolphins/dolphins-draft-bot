GO ?= go
COVERAGE_MIN ?= 85
REMOTE ?=
SSH_OPTS ?=
SERVICE_NAME ?= draft-bot
LOCAL_LOG_DIR ?= ./logs
JOURNAL_MINUTES ?= 180

.PHONY: test coverage lint ci fmt-check logs-download

test:
	$(GO) test ./...

coverage:
	$(GO) test $$($(GO) list ./... | grep -v /cmd/) -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out | tee coverage.txt
	@pct=$$($(GO) tool cover -func=coverage.out | awk '/^total:/ {gsub("%", "", $$3); print $$3}'); \
	awk "BEGIN {if ($$pct < $(COVERAGE_MIN)) {print \"coverage below threshold: \" $$pct \"% < $(COVERAGE_MIN)%\"; exit 1}}"

GOLANGCI_LINT_VERSION ?= v2.14.0

lint:
	command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

fmt-check:
	@test -z "$$($(GO) fmt ./...)" || (echo "go fmt made changes" && exit 1)

ci: fmt-check test coverage lint

logs-download:
	@test -n "$(REMOTE)" || (echo "Uso: make logs-download REMOTE=user@host [JOURNAL_MINUTES=180] [LOCAL_LOG_DIR=./logs] [SERVICE_NAME=draft-bot] [SSH_OPTS='-i ~/.ssh/key']" && exit 1)
	@mkdir -p "$(LOCAL_LOG_DIR)"
	@ts=$$(date +%Y%m%d-%H%M%S); \
	base="$(LOCAL_LOG_DIR)/$(SERVICE_NAME)-$$ts"; \
	echo "Descargando logs desde $(REMOTE)..."; \
	ssh $(SSH_OPTS) "$(REMOTE)" "sudo cat /var/log/$(SERVICE_NAME)/out.log 2>/dev/null || true" > "$$base-out.log"; \
	ssh $(SSH_OPTS) "$(REMOTE)" "sudo cat /var/log/$(SERVICE_NAME)/err.log 2>/dev/null || true" > "$$base-err.log"; \
	ssh $(SSH_OPTS) "$(REMOTE)" "sudo journalctl -u $(SERVICE_NAME) --since '-$(JOURNAL_MINUTES) min' --no-pager 2>/dev/null || true" > "$$base-journal.log"; \
	echo "Logs guardados en:"; \
	echo "  $$base-out.log"; \
	echo "  $$base-err.log"; \
	echo "  $$base-journal.log"
