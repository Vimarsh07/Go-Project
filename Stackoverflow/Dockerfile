# syntax=docker/dockerfile:1
FROM golang:1.21.4

# Set environment variables
ENV PORT 8080
ENV HOSTDIR 0.0.0.0

# Expose the port the app runs on
EXPOSE 8080

# Set the working directory in the container
WORKDIR /app

# Copy Go module files and install dependencies
COPY go.mod ./
COPY go.sum ./
RUN go mod tidy

# Copy the entire project
COPY . ./

# Copy Prometheus configuration file
COPY prometheus.yml /etc/prometheus/prometheus.yml

# Build the Go application
RUN go build -o /main

# Command to run the compiled binary
CMD [ "/main" ]

# You may also need to add a command to start Prometheus, depending on your setup
