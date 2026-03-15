# Stage 1: build web
FROM node:22-alpine AS web
ARG VITE_API_URL=
ENV VITE_API_URL=$VITE_API_URL
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: build kb-server
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./internal/ui/static
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o kb-server ./cmd/kb-server

# Stage 3: minimal runtime
FROM alpine:3.19
RUN apk add --no-cache git openssh-client ca-certificates
RUN adduser -D -g "" kb
USER kb
WORKDIR /data
EXPOSE 8080
ENTRYPOINT ["/kb-server"]
COPY --from=builder /app/kb-server /kb-server
