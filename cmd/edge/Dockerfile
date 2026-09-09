# Build
FROM golang:1.26.5 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /edge ./cmd/edge

# Final image
FROM alpine:3.20
COPY --from=builder /edge /edge
EXPOSE 8001
ENTRYPOINT ["/edge"]