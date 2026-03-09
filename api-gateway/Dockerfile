FROM golang:1.22-alpine

WORKDIR /app

# Needed because api-gateway imports user-service/pkg/... via replace in go.mod
COPY api-gateway/go.mod api-gateway/go.sum /app/api-gateway/
COPY user-service/go.mod user-service/go.sum /app/user-service/

WORKDIR /app/api-gateway
RUN go mod download

WORKDIR /app
COPY api-gateway /app/api-gateway
COPY user-service /app/user-service

WORKDIR /app/api-gateway
RUN go build -mod=mod -o app ./cmd/api

EXPOSE 8080
CMD ["./app"]
