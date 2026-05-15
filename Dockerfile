# Stage 1: build web
FROM node:22-alpine AS web
ARG VITE_API_URL=
ENV VITE_API_URL=$VITE_API_URL
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: build kb
FROM golang:1.25-alpine AS builder
# Уникален на образ: ETag для embed-статики (If-None-Match иначе 304 с прежними бандлами).
ARG BUILD_ID=
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./internal/ui/static
RUN BID="${BUILD_ID}"; \
  if [ -z "$BID" ]; then BID="$(date -u +%Y%m%d%H%M%S)-local"; fi; \
  CGO_ENABLED=0 go build -ldflags="-s -w -X github.com/strider2038/knowledge-db/internal/ui.BuildID=${BID}" -o kb ./cmd/kb

# Stage 3: minimal runtime
FROM alpine:3.19
RUN apk add --no-cache git openssh-client ca-certificates curl bash libstdc++
COPY --from=web /usr/local/bin/node /usr/local/bin/node
RUN adduser -D -g "" kb
USER kb
ENV PATH="/home/kb/.local/bin:${PATH}"
RUN curl https://cursor.com/install -fsS | bash
RUN if [ -d "/home/kb/.local/share/cursor-agent/versions" ]; then \
      find /home/kb/.local/share/cursor-agent/versions -type f -name node -exec sh -c '\
        for f do rm -f "$f"; ln -s /usr/local/bin/node "$f"; done' sh {} +; \
    fi
WORKDIR /data
EXPOSE 8080
ENTRYPOINT ["/kb", "serve"]
COPY --from=builder /app/kb /kb
