# Step 1: Use the official Go image as the base
FROM golang:1.24.1 AS builder

# Step 2: Set working directory
WORKDIR /app

# Step 3: Copy Go source code into the container
COPY . .

# Step 4: Download dependencies
RUN go mod tidy

# Step 5: Build the Go application
RUN go build -o server ./cmd/server/main.go

# Step 6: Use a minimal base image
FROM debian:latest

# Step 7: Install Python
RUN apt-get update && apt-get install -y python3

# Step 8: Set the working directory
WORKDIR /app

# Step 9: Copy built executable and Python script
COPY --from=builder /app/server /app/server
COPY code.py /app/code.py

# Step 10: Expose port 8080
EXPOSE 8080

# Step 11: Start the server
CMD ["/app/server"]
