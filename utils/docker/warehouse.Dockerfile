FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

COPY src/warehouse/frontend/package.json src/warehouse/frontend/package-lock.json* ./
RUN npm install

COPY src/warehouse/frontend/ .
RUN npm run build

FROM golang:1.23-alpine AS go-builder

WORKDIR /build

COPY src/warehouse/go.mod src/warehouse/go.sum ./
RUN GOPROXY=https://proxy.golang.org,direct go mod download

COPY src/warehouse/ .

RUN CGO_ENABLED=0 GOOS=linux GOPROXY=https://proxy.golang.org,direct go build -mod=mod -a -installsuffix cgo -o warehouse ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=go-builder /build/warehouse .
COPY --from=go-builder /build/migrations ./migrations
COPY --from=frontend-builder /frontend/../static ./static

EXPOSE 8082

CMD ["./warehouse"]