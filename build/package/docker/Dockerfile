# syntax=docker/dockerfile:experimental
FROM docker.mirror.hashicorp.services/golang:1.15 as builder

ENV GOBIN=/go/bin
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GO111MODULE=on

WORKDIR /go/src/github.com/hashicorp/inclusify

COPY go.mod go.mod
COPY go.sum go.sum

# Download Go modules
RUN go mod download

COPY ./ .
RUN go build -ldflags "" -o inclusify ./cmd/inclusify

FROM docker.mirror.hashicorp.services/alpine:3.12
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=builder \
    # FROM
    /go/src/github.com/hashicorp/inclusify/inclusify \
    # TO
    /usr/local/bin/inclusify

CMD ["inclusify", "--version"]