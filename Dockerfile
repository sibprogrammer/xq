FROM golang:1.24 AS builder

ARG CGO_ENABLED=0

WORKDIR /opt

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build

FROM ubuntu:22.04

COPY --from=builder /opt/xq /usr/local/bin/xq

ENTRYPOINT ["bash"]
