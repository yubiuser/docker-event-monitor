# syntax=docker/dockerfile:1

FROM golang:1.24-alpine3.22 AS builder
ARG GIT_COMMIT
ARG GIT_BRANCH
ARG GIT_VERSION
ARG GIT_DATE
ARG GIT_TAG

RUN apk add --no-cache \
                git \
                make

# Download dependencies first to cache them
WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN go mod download

# Copy the source code and build
COPY /src ./
RUN make build

FROM scratch AS deploy
COPY --from=builder /app/docker-event-monitor docker-event-monitor
# this pulls directly from the upstream image, which already has ca-certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/docker-event-monitor"]
