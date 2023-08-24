FROM alpine:3.18
RUN apk add --no-cache \
        bash \        
        curl \
        jq

COPY docker-event-monitor.sh ./docker-event-monitor.sh

CMD ["/bin/bash", "docker-event-monitor.sh"]
