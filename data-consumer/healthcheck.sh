#!/bin/bash

# Retry counter
MAX_RETRIES=20
RETRY_INTERVAL=5

# Function to check if Kafka is available
check_kafka() {
  nc -vz ${KAFKA_BOOTSTRAP_HOST} ${KAFKA_BOOTSTRAP_PORT} > /dev/null 2>&1
  return $?
}

# Wait until Kafka is available
for ((i = 1; i <= MAX_RETRIES; i++)); do
  if check_kafka; then
    echo "Kafka is available. Proceeding..."
    exit 0
  fi
  echo "Kafka is not available. Retrying in $RETRY_INTERVAL seconds..."
  sleep $RETRY_INTERVAL
done

echo "Kafka is still not available after $MAX_RETRIES retries. Exiting..."
exit 1

