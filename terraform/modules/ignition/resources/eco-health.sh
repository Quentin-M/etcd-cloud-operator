#!/bin/bash

NIC=$(/usr/bin/awk '$2 == 00000000 { print $1 }' /proc/net/route)
IP=$(/usr/bin/ip -4 addr show ${NIC} | /usr/bin/grep -oP '(?<=inet\s)\d+(\.\d+){3}')

i=0
while true; do
    /opt/bin/e --endpoints=${IP}:2379 endpoint --dial-timeout=500ms --command-timeout=500ms health 2>&1 | grep "successfully" > /dev/null
    if [ $? -eq 0 ]; then
        echo "ECO is healthy"
        i=0
        sleep 60
        continue
    fi
    if [ $i -gt 30 ]; then
        echo "ECO has been unhealthy for the past 900 seconds, shutting down"
        shutdown -h now
        tail -f /dev/null
    fi
    (( i++ ))
    echo "ECO is unhealthy for the ${i}th time (60s interval)"
    sleep 59
done
