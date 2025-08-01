# build stage
FROM golang:1.24-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Cache build and go modules
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/main.go
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o /app/webhook ./webhook_receiver.go

# final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/webhook .
COPY --from=builder /app/.env .
COPY --from=builder /app/internal/migrations ./internal/migrations
EXPOSE 8080
EXPOSE 9090
CMD ["sh", "-c", "./main & ./webhook"]