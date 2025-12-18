FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

COPY src/ti-server/frontend/package.json src/ti-server/frontend/package-lock.json* ./
RUN npm install --no-audit

COPY src/ti-server/frontend/ .

RUN npm run build

FROM golang:1.23-alpine AS go-builder

WORKDIR /build

COPY src/ti-server/go.mod src/ti-server/go.sum ./
RUN GOPROXY=https://proxy.golang.org,direct go mod download

COPY src/ti-server/ .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ti-server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=go-builder /build/ti-server .
COPY --from=go-builder /build/migrations ./migrations
COPY --from=frontend-builder /frontend/../static ./static

EXPOSE 8080

CMD ["./ti-server"]