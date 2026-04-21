# syntax=docker/dockerfile:1.7

FROM node:20.19-bookworm-slim AS web-builder
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build && npm prune --production

FROM golang:1.25.9-bookworm AS go-builder
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
WORKDIR /src
ENV CGO_ENABLED=0 \
    GOFLAGS=-mod=readonly
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -buildvcs=true \
  -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
  -o /out/viaduct ./cmd/viaduct

FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
LABEL org.opencontainers.image.title="Viaduct" \
      org.opencontainers.image.description="Hypervisor-agnostic workload migration and lifecycle management platform" \
      org.opencontainers.image.source="https://github.com/eblackrps/viaduct" \
      org.opencontainers.image.documentation="https://github.com/eblackrps/viaduct/tree/main/docs/operations/docker.md" \
      org.opencontainers.image.vendor="Eric Black" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.url="https://github.com/eblackrps/viaduct"

WORKDIR /opt/viaduct
COPY --from=go-builder /out/viaduct /viaduct
COPY --from=web-builder /src/web/dist /opt/viaduct/web

ENV HOME=/tmp \
    VIADUCT_WEB_DIR=/opt/viaduct/web

EXPOSE 8080
VOLUME ["/var/lib/viaduct"]
USER 65532:65532
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD ["/viaduct", "healthcheck", "--url", "http://127.0.0.1:8080/healthz", "--timeout", "5s"]

ENTRYPOINT ["/viaduct"]
CMD ["serve-api", "--host", "0.0.0.0", "--port", "8080"]
