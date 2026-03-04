FROM golang:1.24-alpine

WORKDIR /app

# Needed because api-gateway imports user-service/pkg/... via replace in go.mod
COPY api-gateway /app/api-gateway
COPY user-service /app/user-service

WORKDIR /app/api-gateway
RUN go build -o app ./cmd/api

EXPOSE 8080
CMD ["./app"]
