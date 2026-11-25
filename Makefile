# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
SHELL := /bin/bash
.DEFAULT_GOAL := help
export GOFLAGS := -buildvcs=false
export GOTOOLCHAIN := go1.27.4
export GOCACHE ?= /tmp/provenant-passage-go-cache
export GOLANGCI_LINT_CACHE ?= /tmp/provenant-passage-lint-cache
IMPORTER_SOURCE ?= .cache/provenant-import
BINE_PATH = $(shell GOTOOLCHAIN=go1.27.4 GOFLAGS=-buildvcs=false GOCACHE=/tmp/provenant-passage-go-cache go tool bine path)
export PATH := $(BINE_PATH):$(PATH)

.PHONY: help tools importer build fmt lint test test-race vet workflowcheck gen-enums gen-ent gen-goa gen-ui-client gen-mock atlas-hash mod-tidy
help:
	@echo 'tools importer build fmt lint test test-race vet workflowcheck gen-enums gen-ent gen-goa gen-ui-client gen-mock atlas-hash mod-tidy'
tools:
	go tool bine sync
tool-%:
	go tool bine get $*
importer:
	mkdir -p bin
	cd "$(IMPORTER_SOURCE)" && CGO_ENABLED=0 go build -mod=readonly -buildvcs=false -trimpath -ldflags='-s -w' -o "$(CURDIR)/bin/provenant-import" ./cmd/provenant-import
build: importer
	bash hack/build_dist.sh -o bin/provenant-server ./cmd/provenant-server
	bash hack/build_dist.sh -o bin/provenant-migrate ./cmd/migrate
	bash hack/build_dist.sh -o bin/provenant-reset ./cmd/provenant-reset
fmt: tool-golangci-lint
	golangci-lint fmt
lint: tool-golangci-lint
	golangci-lint run --timeout=5m
test:
	go test ./cmd/... ./internal/...
test-race:
	go test -race ./cmd/... ./internal/...
vet:
	go vet ./cmd/... ./internal/...
workflowcheck: tool-workflowcheck
	workflowcheck ./internal/workflow/...
gen-enums: tool-go-enum
	go-enum --names --template=$(CURDIR)/hack/make/enums.tmpl --nocomments -f internal/enums/status.go -f internal/enums/profile.go
gen-ent:
	go tool ent generate ./internal/persistence/ent/schema --feature sql/versioned-migration,sql/upsert --target=./internal/persistence/ent/db
gen-goa:
	go tool goa gen github.com/jraddaoui/provenant-passage/internal/api/design -o internal/api
gen-ui-client:
	docker run --rm --user $$(id -u):$$(id -g) --volume $(CURDIR):/local openapitools/openapi-generator-cli:v7.24.0 generate --input-spec /local/internal/api/gen/http/openapi3.yaml --generator-name typescript-fetch --output /local/ui/src/openapi-generator/ --global-property apiDocs=false,modelDocs=false -p generateAliasAsModel=false,removeEnumValuePrefix=false,supportsES6=true,withInterfaces=true
gen-mock:
	go generate ./internal/...
atlas-hash: tool-atlas
	atlas migrate hash --dir=file://internal/db/migrations --dir-format=golang-migrate
mod-tidy:
	go mod tidy
