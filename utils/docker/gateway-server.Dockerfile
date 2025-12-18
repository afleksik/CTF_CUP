FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

COPY src/gateway-server/frontend/package.json src/gateway-server/frontend/package-lock.json* ./
RUN npm install

COPY src/gateway-server/frontend/ .

RUN npm run build

FROM golang:1.23-alpine AS go-builder

WORKDIR /build

COPY src/gateway-server/go.mod src/gateway-server/go.sum ./
RUN GOPROXY=https://proxy.golang.org,direct go mod download

COPY src/gateway-server/ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gateway-server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=go-builder /build/gateway-server .

COPY --from=go-builder /build/migrations ./migrations

COPY --from=frontend-builder /frontend/../static ./static

EXPOSE 8000

CMD ["./gateway-server"]