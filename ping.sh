#!/bin/bash

LOG_DIR="logs/"
PING_RESULTS_FILE="ping_results.log"

# Function to monitor logs for "Gateway Timeout"
monitor_logs() {
    while true; do
        for file in "$LOG_DIR"info.*.log; do
            if grep -q "Gateway Timeout" "$file"; then
                ping_hosts
            fi
        done
        sleep 10 # Check every 10 seconds
    done
}

# Function to ping hosts and log results
ping_hosts() {
    for host in "www.google.com" "api.telegram.org"; do
        result=$(ping -c 4 "$host" 2>&1)
        log_ping_results "$host" "$result"
    done
}

# Function to log ping results
log_ping_results() {
    local host="$1"
    local result="$2"
    echo "Ping result for $host:" >> "$PING_RESULTS_FILE"
    echo "$result" >> "$PING_RESULTS_FILE"
    echo "" >> "$PING_RESULTS_FILE"
}

# Start monitoring logs
monitor_logs


