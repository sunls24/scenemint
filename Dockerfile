# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26
ARG BUN_VERSION=1

FROM --platform=$BUILDPLATFORM oven/bun:${BUN_VERSION}-alpine AS web-builder
WORKDIR /src/web

COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile

COPY web/ ./
RUN bun run build

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS go-builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

ARG TARGETOS
ARG TARGETARCH
RUN target_os=${TARGETOS:-linux}; \
  target_arch=${TARGETARCH:-$(go env GOARCH)}; \
  CGO_ENABLED=0 GOOS=${target_os} GOARCH=${target_arch} go build -trimpath -ldflags="-s -w" -o /out/scenemint ./cmd

FROM alpine:3.22
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
  && addgroup -S scenemint \
  && adduser -S scenemint -G scenemint \
  && chown -R scenemint:scenemint /app

COPY --from=go-builder /out/scenemint ./scenemint
COPY --from=web-builder /src/web/dist ./web/dist
RUN mkdir -p /app/data \
  && chown -R scenemint:scenemint /app/data

USER scenemint

ENV HOST=0.0.0.0
ENV PORT=3000
ENV CHATGPT2API_IMAGE_MODEL=gpt-image-2
ENV CHATGPT2API_PROMPT_MODEL=gpt-5.5

EXPOSE 3000

CMD ["./scenemint"]
