FROM golang:1.24

WORKDIR /app
COPY main.go .
RUN go mod init post-service
RUN go mod tidy
RUN go build -o post-service
EXPOSE 5000
ENTRYPOINT ["./post-service"]
