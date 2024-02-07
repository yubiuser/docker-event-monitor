ARG alpine_version=3.19
ARG golang_version=1.21

FROM golang:${golang_version}-alpine${alpine_version} as builder
ARG GIT_COMMIT
ARG GIT_BRANCH
ARG GIT_VERSION
ARG GIT_DATE
ARG GIT_TAG

RUN apk add --no-cache \
                git \
                make

COPY /src /src
WORKDIR /src
RUN make build

FROM scratch as deploy
COPY --from=builder /src/docker-event-monitor docker-event-monitor
# the tls certificates:
# this pulls directly from the upstream image, which already has ca-certificates:
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/docker-event-monitor"]
