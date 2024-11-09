FROM --platform=${BUILDPLATFORM} golang:1.17.0-alpine AS build
RUN apk update
RUN apk add --no-cache git

RUN go env -w GO111MODULE=on

ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

WORKDIR /src/github.com/transifex/cli
