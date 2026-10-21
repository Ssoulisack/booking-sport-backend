FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./
COPY go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Use a smaller image for the final stage
FROM alpine:latest

# Set timezone
ENV TZ=Asia/Bangkok
# Install tzdata for timezone support
RUN apk add --no-cache tzdata

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/configs ./configs

# Expose the application port
EXPOSE 3000

# Command to run the executable
CMD ["./main"]
