# Single-instance image: the Go API also serves the built GUI.
#   docker build -t upstonk .
#   docker run -p 8080:8080 --env-file server/.env upstonk

# 1. GUI
FROM node:24-alpine AS gui
WORKDIR /gui
COPY gui/package.json gui/package-lock.json ./
RUN npm ci
COPY gui/ ./
RUN npm run build

# 2. API server (static binary)
FROM golang:1.26-alpine AS server
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/upstonk ./cmd/server

# 3. Runtime
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -u 10001 upstonk
COPY --from=server /out/upstonk /app/upstonk
COPY --from=gui /gui/dist /app/web
USER upstonk
# Listens on $PORT when the host sets it (Render does), else :8080.
ENV WEB_DIR=/app/web
EXPOSE 8080
ENTRYPOINT ["/app/upstonk"]
