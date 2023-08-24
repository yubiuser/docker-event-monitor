#!/bin/bash

# The following variables need to be set via docker environment variables
# PUSHOVER_TOKEN, PUSHOVER_USER and FILTERS

pushover(){
    local titel=$1 message=$2

    curl -s -F "token=${PUSHOVER_TOKEN}" \
    -F "user=${PUSHOVER_USER}" \
    -F "title=$titel" \
    -F "message=$message" https://api.pushover.net/1/messages.json\
    -o /dev/null # omit response
}


# Monitor all Docker events
curl    --no-buffer\
        --silent\
        --request GET "http://localhost/events"\
        --unix-socket /var/run/docker.sock \
        --get --data-urlencode "${FILTERS}" | while read -r event
        do
            mapfile -t data <<<"$(echo "${event}" | jq --raw-output '.id, .status, .from, .time')"
            container_id="${data[0]:0:8}" #limit container id to 8 characters
            status="${data[1]}"
            image="${data[2]##*/}" # remove everything before the final '/' if exists
            time="$(date -d @"${data[3]}" '+%x %X')" # convert unix timestamp to local date + time
            echo "Container ${container_id} from image ${image} with new status: ${status}" # log to docker logs
            pushover "Container ${container_id} from image ${image}" "${time}  Status: ${status}"
        done

