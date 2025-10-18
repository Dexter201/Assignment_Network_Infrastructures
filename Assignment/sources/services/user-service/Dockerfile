FROM golang:1.24

WORKDIR /app
COPY main.go .
RUN go mod init user-service
RUN go mod tidy
RUN go build -o user-service
EXPOSE 5000
ENTRYPOINT ["./user-service"]
