FROM golang:1.21.6-bullseye AS build

ENV DEBIAN_FRONTEND="noninteractive"
ENV TZ="Etc/UTC"
ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="1"

WORKDIR /usr/src/stream

COPY go.mod go.sum ./

RUN go mod download && \
  go mod verify

COPY pkg ./pkg

COPY cmd/stream ./cmd/stream

RUN go build \
        -v \
        -trimpath \
        -tags timetzdata \
        -o /stream \
        ./cmd/stream

FROM debian:bullseye-slim

ENV DEBIAN_FRONTEND="noninteractive"
ENV TZ="Etc/UTC"

RUN apt-get update && apt-get install --yes \
  dumb-init \
  ca-certificates

WORKDIR /stream
COPY --from=build /stream /usr/bin/stream

CMD ["/usr/bin/stream"]
