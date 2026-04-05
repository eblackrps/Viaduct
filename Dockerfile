FROM node:25-bookworm-slim AS web-build
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.24-bookworm AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=container -X main.commit=container -X main.date=unknown" -o /out/viaduct ./cmd/viaduct

FROM debian:bookworm-slim
RUN useradd --create-home --home-dir /home/viaduct --shell /usr/sbin/nologin viaduct
WORKDIR /opt/viaduct

COPY --from=go-build /out/viaduct /usr/local/bin/viaduct
COPY --from=web-build /src/web/dist /opt/viaduct/web
COPY configs /opt/viaduct/configs
COPY docs /opt/viaduct/docs
COPY examples /opt/viaduct/examples

ENV HOME=/home/viaduct
EXPOSE 8080
USER viaduct

ENTRYPOINT ["viaduct", "serve-api", "--port", "8080"]
