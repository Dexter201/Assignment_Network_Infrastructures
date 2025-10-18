# !! The comments set are only used to help me learn more about docker. 
# !! By the fact that i am not profficiant with it yet, these comments help me not to forget what each command does.

FROM golang:1.24

# Set the working directory inside the container to `/app`.
# All subsequent commands (e.g., COPY, RUN) will be executed in this directory.
WORKDIR /app

# Copy `go.mod` and `go.sum` files to the container.
# These files define the project's dependencies and their versions.
# This means if only your source code changes (not dependencies).
COPY go.mod go.sum ./

# Download all dependencies listed in `go.mod`.
RUN go mod download

# Copy the rest of the application code (including `main.go`) to the container.
# This step copies all files from your local directory to the `/app` directory in the container.
COPY . .

# Build the Go application.
# This compiles your Go code into a binary named `user-service`.
RUN go build -o user-service

# Inform Docker that the container will listen on port 5000 at runtime.
# This is purely informational and does not actually publish the port.
# You still need to use `-p 5000:5000` when running the container to map the port.
EXPOSE 5000

# Specify the command to run when the container starts.
# This command executes the `user-service` binary, which starts your application.
ENTRYPOINT ["./user-service"]
