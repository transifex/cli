FROM --platform=${BUILDPLATFORM} golang:1.16.0-alpine AS build
RUN apk update
RUN apk add --no-cache git

RUN go env -w GO111MODULE=on

WORKDIR /src

COPY /src/go.mod ./
COPY /src/go.sum ./

RUN go mod download

COPY /src ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /bin/cli .

FROM scratch AS bin
COPY --from=build /bin/cli /

