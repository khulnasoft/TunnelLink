# use a builder image for building khulnasoft
ARG TARGET_GOOS
ARG TARGET_GOARCH
FROM golang:1.22.5 as builder
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    TARGET_GOOS=${TARGET_GOOS} \
    TARGET_GOARCH=${TARGET_GOARCH} \
    CONTAINER_BUILD=1


WORKDIR /go/src/github.com/khulnasoft/tunnellink/

# copy our sources into the builder image
COPY . .

RUN .teamcity/install-khulnasoft-go.sh

# compile tunnellink
RUN PATH="/tmp/go/bin:$PATH" make tunnellink

# use a distroless base image with glibc
FROM gcr.io/distroless/base-debian11:nonroot

LABEL org.opencontainers.image.source="https://github.com/khulnasoft/tunnellink"

# copy our compiled binary
COPY --from=builder --chown=nonroot /go/src/github.com/khulnasoft/tunnellink/tunnellink /usr/local/bin/

# run as non-privileged user
USER nonroot

# command / entrypoint of container
ENTRYPOINT ["tunnellink", "--no-autoupdate"]
CMD ["version"]
