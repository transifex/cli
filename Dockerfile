FROM golang:1.16.0-alpine AS build
RUN apk update
RUN apk add --no-cache git

RUN go env -w GO111MODULE=on

WORKDIR /src

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o cli .

CMD ["sh"]
