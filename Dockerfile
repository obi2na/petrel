#stage 1 - Build the Go binary
FROM golang:1.24.4 AS builder

WORKDIR /app

# Cache go mod downloads
COPY go.mod go.sum ./
RUN go mod download

#Copy all source files
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2 - Final minimal image
FROM gcr.io/distroless/base-debian11

COPY --from=builder /app/server /server

EXPOSE 8080
ENTRYPOINT ["/server"]