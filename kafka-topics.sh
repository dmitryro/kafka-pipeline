#!/bin/bash

# Kafka broker URL
BROKER=$2
TOPIC=$4
PARTITIONS=$6
REPLICATION=$8

# Check if correct arguments are passed
if [[ $# -lt 8 ]]; then
    echo "Usage: kafka-topics.sh --create --topic <topic_name> --partitions <num_partitions> --replication-factor <num_replicas> --bootstrap-server <kafka_broker>"
    exit 1
fi

# Create the Kafka topic
kafka-topics --create \
    --topic "$TOPIC" \
    --partitions "$PARTITIONS" \
    --replication-factor "$REPLICATION" \
    --bootstrap-server "$BROKER" \
    --if-not-exists

# Display a confirmation message
echo "Topic '$TOPIC' created successfully with $PARTITIONS partitions and $REPLICATION replication factor."

