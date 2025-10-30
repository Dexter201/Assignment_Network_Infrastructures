FROM golang:1.24
WORKDIR /app
ENV GO111MODULE=on

COPY go.mod go.sum ./
RUN go mod tidy
RUN go mod download

COPY . . 

RUN go build -o load-balancer
EXPOSE 8080
ENTRYPOINT ["./load-balancer"]