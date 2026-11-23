# syntax=docker/dockerfile:1.7
#
# Multi-stage build for Thoth's Collection.
#
#   target "api" -> Go gRPC/gateway server (+ the scryfall sync CLI)
#   target "web" -> nginx serving the built React SPA and proxying to "api"
#   target "dev" -> devcontainer toolchain (Go, Node, buf, sqlc, air, lint)
#   targets "test-go" / "test-web" -> CI checks (vet + tests, tsc + eslint)
#
# Generated code (buf + sqlc) is gitignored, so it's produced in the "codegen"
# stage and shared by both the Go and frontend builds.

ARG GO_VERSION=1.27
ARG NODE_VERSION=24
ARG NGINX_VERSION=1.29
ARG BUF_VERSION=1.73.0
ARG SQLC_VERSION=1.30.0

FROM bufbuild/buf:${BUF_VERSION} AS buf
FROM sqlc/sqlc:${SQLC_VERSION} AS sqlc
FROM node:${NODE_VERSION}-bookworm-slim AS node

# ---------------------------------------------------------------------------
# Dev image for .devcontainer. The repo is bind-mounted at runtime, so nothing
# from the source tree is copied in here.
# ---------------------------------------------------------------------------
FROM golang:${GO_VERSION}-bookworm AS dev
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=1000

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
        sudo git make less default-mysql-client \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd --gid ${USER_GID} ${USERNAME} \
    && useradd --uid ${USER_UID} --gid ${USER_GID} -m -s /bin/bash ${USERNAME} \
    && echo "${USERNAME} ALL=(root) NOPASSWD:ALL" > /etc/sudoers.d/${USERNAME} \
    && chmod 0440 /etc/sudoers.d/${USERNAME} \
    # Let the non-root user own GOPATH (module + build caches, go install).
    && mkdir -p /go/pkg/mod /home/${USERNAME}/.cache \
    && chown -R ${USER_UID}:${USER_GID} /go /home/${USERNAME}/.cache

COPY --from=buf /usr/local/bin/buf /usr/local/bin/buf
COPY --from=sqlc /workspace/sqlc /usr/local/bin/sqlc
COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s ../lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s ../lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

USER ${USERNAME}
RUN go install github.com/air-verse/air@latest \
    && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest \
    && go clean -cache

# ---------------------------------------------------------------------------
# Code generation: protobuf (Go + TS) and sqlc
# ---------------------------------------------------------------------------
FROM golang:${GO_VERSION}-bookworm AS codegen
COPY --from=buf /usr/local/bin/buf /usr/local/bin/buf
COPY --from=sqlc /workspace/sqlc /usr/local/bin/sqlc
WORKDIR /src
COPY proto/ proto/
COPY db/ db/
COPY internal/db/queries/ internal/db/queries/
# buf.gen.yaml uses remote plugins, so this step needs network access.
RUN --mount=type=cache,target=/root/.cache/buf \
    cd proto && buf generate
RUN sqlc generate --file db/sqlc.yaml

# ---------------------------------------------------------------------------
# Go build
# ---------------------------------------------------------------------------
FROM golang:${GO_VERSION}-bookworm AS api-build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY db/ db/
COPY --from=codegen /src/internal/gen/ internal/gen/
COPY --from=codegen /src/internal/db/ internal/db/
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/thoth ./cmd/server && \
    go build -trimpath -ldflags="-s -w" -o /out/sync ./cmd/sync && \
    mkdir -p /out/cache

# ---------------------------------------------------------------------------
# Go checks (CI): `docker buildx build --target test-go .`
# ---------------------------------------------------------------------------
FROM api-build AS test-go
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go vet ./... && go test ./...

# ---------------------------------------------------------------------------
# API runtime image
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS api
COPY --from=api-build /out/thoth /out/sync /app/
# Pre-create the scryfall cache dir so a named volume mounted here inherits
# nonroot ownership.
COPY --from=api-build --chown=65532:65532 /out/cache /var/cache/thoth
EXPOSE 8080 9090
ENTRYPOINT ["/app/thoth"]
CMD ["--http-addr=:8080", "--grpc-addr=:9090"]

# ---------------------------------------------------------------------------
# Frontend build
# ---------------------------------------------------------------------------
FROM node AS web-build
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json frontend/.npmrc ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY frontend/ ./
COPY --from=codegen /src/frontend/src/gen/ src/gen/
RUN npm run build

# ---------------------------------------------------------------------------
# Frontend checks (CI): `docker buildx build --target test-web .`
# web-build has already type-checked (tsc) and built; this adds eslint.
# ---------------------------------------------------------------------------
FROM web-build AS test-web
RUN npm run lint

# ---------------------------------------------------------------------------
# nginx runtime image
# ---------------------------------------------------------------------------
FROM nginx:${NGINX_VERSION}-alpine AS web
COPY docker/nginx/thoth-locations.conf /etc/nginx/snippets/thoth-locations.conf
COPY docker/nginx/http.conf /etc/nginx/conf.d/default.conf
COPY docker/nginx/tls.conf.template /etc/nginx/thoth/tls.conf.template
COPY --chmod=755 docker/nginx/40-thoth-tls.sh /docker-entrypoint.d/40-thoth-tls.sh
COPY --from=web-build /src/frontend/dist/ /usr/share/nginx/html/
