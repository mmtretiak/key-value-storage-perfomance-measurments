# Build the application
FROM golang:1.23.2-bookworm AS build

WORKDIR /go/src
COPY .. .
RUN go mod vendor -v
RUN env GOOS=linux GOARCH=amd64 go build -mod vendor -x ./cmd/natsVsRedis

# Build the deploy container
FROM debian:bookworm-slim

# Install basic binaries
RUN apt-get -y update && apt-get install -y ca-certificates

ARG USER_UID=999
ARG GROUP_GID=999
ARG UGNAME=siden

RUN groupadd -g ${USER_UID} siden \
    && \
    useradd --no-log-init -r -u ${GROUP_GID} -g ${UGNAME} ${UGNAME} \
    && \
    mkdir -p /siden \
    && \
    chmod 555 /siden \
    && \
    chown ${USER_UID}:${GROUP_GID} /siden

WORKDIR /siden/
USER siden

COPY --from=build /go/src/natsVsRedis /siden/

ENTRYPOINT ["/siden/natsVsRedis"]
