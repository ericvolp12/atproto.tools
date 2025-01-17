FROM golang:1.23-bullseye AS build

ENV DEBIAN_FRONTEND="noninteractive"
ENV TZ="Etc/UTC"
ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="1"

WORKDIR /usr/src/plc

COPY go.mod go.sum ./

RUN go mod download && \
    go mod verify

COPY pkg ./pkg

COPY cmd/plc ./cmd/plc

RUN go build \
    -v \
    -trimpath \
    -tags timetzdata \
    -o /plc \
    ./cmd/plc

FROM debian:bullseye-slim

ENV DEBIAN_FRONTEND="noninteractive"
ENV TZ="Etc/UTC"

RUN apt-get update && apt-get install --yes \
    dumb-init \
    ca-certificates

WORKDIR /plc
COPY --from=build /plc /usr/bin/plc

CMD ["/usr/bin/plc"]
