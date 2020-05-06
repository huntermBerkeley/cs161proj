FROM golang:alpine

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64

RUN mkdir /app

RUN apk add build-base


ADD . /app/
WORKDIR /app
RUN go build -o main .


CMD ["/app/main"]
