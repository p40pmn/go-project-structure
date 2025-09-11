# Start from golang base image as building stage
FROM golang:1.25-alpine AS builder

# Set necessary environment variables needed for our image
ENV GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Set the current working directory inside the container
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN apk add --no-cache ca-certificates git tzdata && \
  go mod tidy

# Copy the code into the container
COPY . .

# Build the Go application
RUN go build -ldflags "-s -w -extldflags '-static'" -installsuffix cgo -o /bin/service cmd/main.go

# Use alpine image as runtime
FROM alpine:3.22 AS release

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/service /bin/service

# Command to run 
ENTRYPOINT ["/bin/service"]