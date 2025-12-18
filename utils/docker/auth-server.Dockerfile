FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

COPY src/auth-server/frontend/package.json src/auth-server/frontend/package-lock.json* ./
RUN npm install

COPY src/auth-server/frontend/ .
RUN npm run build

FROM golang:1.23-alpine AS go-builder

WORKDIR /build

COPY src/auth-server/go.mod src/auth-server/go.sum ./
RUN GOPROXY=https://proxy.golang.org,direct go mod download

COPY src/auth-server/ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth-server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=go-builder /build/auth-server .
COPY --from=go-builder /build/migrations ./migrations
COPY --from=frontend-builder /frontend/../static ./static

RUN mkdir -p /app/keys

EXPOSE 8081

CMD ["./auth-server"]