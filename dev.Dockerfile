FROM golang:1.22.5 as builder
ENV GO111MODULE=on \
    CGO_ENABLED=0
WORKDIR /go/src/github.com/khulnasoft/tunnellink/
RUN apt-get update
COPY . .
RUN .teamcity/install-khulnasoft-go.sh
# compile tunnellink
RUN PATH="/tmp/go/bin:$PATH" make tunnellink
RUN cp /go/src/github.com/khulnasoft/tunnellink/tunnellink /usr/local/bin/
