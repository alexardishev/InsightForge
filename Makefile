GO ?= go
GO_DIRS := analytics-data-center

.PHONY: build test test-race test-integration lint fmt fmt-check

## build: собрать все пакеты
build:
	$(GO) build ./...

## test: юнит-тесты
test:
	$(GO) test ./...

## test-race: юнит-тесты с детектором гонок (как в CI)
test-race:
	$(GO) test -race ./...

## test-integration: юнит- и интеграционные тесты (//go:build integration), нужен Docker
test-integration:
	$(GO) test -tags=integration ./...

## lint: статический анализ
lint:
	$(GO) vet ./...

## fmt: отформатировать код
fmt:
	gofmt -w $(GO_DIRS)

## fmt-check: упасть, если код не отформатирован (как в CI).
## На Windows с core.autocrlf=true файлы в рабочей копии с CRLF, и gofmt считает их
## неотформатированными — там эта проверка ложно падает; в CI (Linux) она точная.
fmt-check:
	@unformatted="$$(gofmt -l $(GO_DIRS))"; \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
