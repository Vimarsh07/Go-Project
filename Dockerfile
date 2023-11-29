# Use the official Go image from the Docker Hub
FROM golang:1.21.4

# Copy the local package files to the container's workspace.
ADD . /go/src/myapp

# Setting up working directory
WORKDIR /go/src/myapp

# Install the package.
RUN go mod tidy
RUN go build -o /stackoverflow

# Run the command by default when the container starts.
ENTRYPOINT /stackoverflow

# Document that the service listens on port 8080.
EXPOSE 8080