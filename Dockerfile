# syntax=docker/dockerfile:1.4
# SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
ARG GO_VERSION=1.27.4
FROM golang:${GO_VERSION}-alpine AS build-server
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-buildvcs=false
COPY --link go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY --link cmd/ ./cmd/
COPY --link internal/ ./internal/
COPY --link hack/collect-go-notices.sh /collect-go-notices.sh
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go build -mod=readonly -trimpath -ldflags='-s -w' -o /out/provenant-server ./cmd/provenant-server && \
    go build -mod=readonly -trimpath -ldflags='-s -w' -o /out/provenant-migrate ./cmd/migrate && \
    go build -mod=readonly -trimpath -ldflags='-s -w' -o /out/provenant-reset ./cmd/provenant-reset && \
    sh /collect-go-notices.sh /notices

FROM golang:${GO_VERSION}-alpine AS build-importer
WORKDIR /importer
ENV CGO_ENABLED=0 GOFLAGS=-buildvcs=false
COPY --from=importer-source /go.mod /go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY --from=importer-source /cmd/ ./cmd/
COPY --from=importer-source /internal/ ./internal/
COPY --link hack/collect-go-notices.sh /collect-go-notices.sh
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go build -mod=readonly -buildvcs=false -trimpath -ldflags='-s -w' -o /out/provenant-import ./cmd/provenant-import && \
    sh /collect-go-notices.sh /notices

FROM debian:13-slim AS provenant-server
LABEL org.opencontainers.image.source="https://github.com/jraddaoui/provenant-passage"
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    groupadd --gid 1000 provenant && useradd --uid 1000 --gid 1000 -m provenant
COPY --from=build-server /out/ /usr/local/bin/
COPY --from=build-importer /out/provenant-import /usr/local/bin/provenant-import
COPY --from=build-server /notices/ /usr/share/doc/provenant-passage/dependencies/
COPY --from=build-importer /notices/ /usr/share/doc/provenant-import/dependencies/
COPY LICENSES/ /usr/share/doc/provenant-passage/LICENSES/
COPY DCO /usr/share/doc/provenant-passage/DCO
COPY NOTICE /usr/share/doc/provenant-passage/NOTICE
COPY docs/importer-source-sha256.json /usr/share/doc/provenant-import/source-sha256.json
COPY --from=importer-source /LICENSES/ /usr/share/doc/provenant-import/LICENSES/
COPY --from=importer-source /DCO /usr/share/doc/provenant-import/DCO
COPY provenant.toml /etc/provenant/provenant.toml
USER 1000:1000
EXPOSE 9000 9001
CMD ["provenant-server", "--config", "/etc/provenant/provenant.toml"]
