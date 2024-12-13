
# Real-Time Data Streaming Pipeline with Kafka, Docker, and Go

This project implements a real-time data streaming pipeline using **Apache Kafka**, **Docker**, and **Go**. It involves creating a system to consume, process, and produce data messages in Kafka topics, while ensuring scalability, fault tolerance, and efficient message handling. The pipeline consists of the following components:

1. **Kafka** for message ingestion and distribution.
2. **Docker** for containerization of services.
3. **Go-based consumer service** to process and produce data.
4. **Python-based producer service** for simulating data generation and pushing it to Kafka.

## Overview of the Solution

The solution involves setting up a Kafka consumer in Go that consumes messages from a Kafka topic (`user-login`), processes them based on certain rules, and publishes the processed data to another Kafka topic (`processed-user-login`). Any invalid messages are sent to a Dead Letter Queue (DLQ) topic (`user-login-dlq`). 

### Key Features:
- **Kafka consumer**: Reads messages from a Kafka topic (`user-login`), processes them based on certain checks, and publishes valid messages to the `processed-user-login` topic.
- **Dead Letter Queue (DLQ)**: Invalid messages are sent to the `user-login-dlq` topic.
- **Fault tolerance and retries**: The consumer ensures that messages are processed even in case of temporary issues, using retry logic with exponential backoff.
- **Graceful shutdown**: The application handles shutdown signals to close Kafka consumer and producer connections cleanly.

## Design Choices and Considerations

### 1. **Kafka Topics and Data Flow**
   - **Input Topic**: `user-login` – This is the main topic where messages are consumed from. Messages in this topic are expected to contain user login information in JSON format.
   - **Output Topic**: `processed-user-login` – After processing, valid messages are published to this topic.
   - **Dead Letter Queue (DLQ)**: `user-login-dlq` – Any invalid messages (e.g., missing fields or invalid data) are sent to this DLQ for further inspection.

### 2. **Consumer Logic in Go**
   - The consumer subscribes to the `user-login` topic.
   - It processes each message and validates fields like `UserID`, `AppVersion`, and `DeviceType`. If a message fails validation, it is sent to the DLQ.
   - The consumer uses a worker pool model to handle message processing concurrently, improving throughput.
   - Kafka consumer offsets are managed manually to provide greater control over message acknowledgment, which ensures that messages are not lost in case of a failure.

### 3. **Choice of Kafka, third party libraries and implementaton language** 
   - **Design Priorities**: The design is built with efficiency, scalability, and fault tolerance in mind. 
   - **Go as the programming language**: Go was chosen for its lightweight, asynchronous, and efficient nature. It is well-suited for building scalable and performant systems that interact with Kafka, where message consumption and processing can occur concurrently without blocking other operations. Go's simplicity, fast compilation time, and robust concurrency model (goroutines and channels) make it an ideal choice for this type of real-time data pipeline.
   - **`confluentinc/confluent-kafka-go`**: This library was selected for its minimal configuration, high performance, and strong integration with the Confluent Kafka ecosystem. It provides a reliable, efficient, and straightforward interface for Kafka consumers and producers, making it the most practical choice for this project. Alternatives like `goka`, `kafka-go`, and `Shopify/sarama` could also be used, but `confluent-kafka-go` was chosen due to its direct support for Kafka's native protocol and integration with Kafka.
   - **Kafka as the backbone of the data pipeline**: Kafka is used for handling high throughput of streaming data. Kafka's partitioning model and fault tolerance through replication provide scalability and reliability, ensuring the system can handle large volumes of data without significant performance degradation.

### 4. **Fault Tolerance and Scalability**
   - **Retries**: The consumer implements exponential backoff for retries to avoid overloading the Kafka brokers in case of transient issues.
   - **Concurrency**: A worker pool is used to process multiple messages concurrently, improving throughput and scalability.
   - **Graceful Shutdown**: The consumer listens for termination signals (e.g., SIGTERM) and shuts down Kafka connections cleanly, ensuring no data is lost.

## Running the Project Locally

### Prerequisites
- **Docker** and **Docker Compose** must be installed. 
- **Kafka** and **Zookeeper** will be run as Docker containers.

### Steps to Run the Project

1. Clone this repository:
   ```bash
   git clone git@github.com:dmitryro/kafka-pipeline.git > data_pipeline 
   cd data_pipeline
   ```

2. Build and start the services using Docker Compose:
   ```bash
   docker-compose up --build
   ```

   This command will build all the Docker images and start the following services:
   - **Kafka**: A Kafka broker running on port `9092` (internal) and `29092` (external).
   - **Zookeeper**: A Zookeeper instance used by Kafka for coordination.
   - **Producer Service (Python)**: A producer that generates and sends data to Kafka.
   - **Consumer Service (Go)**: The consumer that processes and publishes data to Kafka topics.

3. After running the above command, the services should be up and running. You can verify this by checking the logs of the consumer:
   ```bash
   docker logs pipeline-consumer
   ```

   You can also use Kafka's `kafka-console-consumer` tool to check the messages in the `processed-user-login` topic:
   ```bash
   kafka-console-consumer --bootstrap-server localhost:29092 --topic processed-user-login --from-beginning
   ```

### Environment Variables

- **LEVEL**: Controls the logging level. Set to `DEBUG` in `.env` for development and `INFO` for production.
- **KAFKA_LISTENER**: The Kafka broker URL for internal communication (e.g., `kafka:9092`).
- **KAFKA_BROKER_URL**: The Kafka broker URL for external communication (e.g., `localhost:29092`).
- **KAFKA_CREATE_TOPICS**: Comma-separated list of Kafka topics to create on startup.
- **KAFKA_ZOOKEEPER_CONNECT**: Connection string for Zookeeper.

See `.env` for all available environment variables and their descriptions.

### Sample .env File

```env
LEVEL=DEBUG
PROJECT_NAME=pipeline
KAFKA_LISTENER=kafka://kafka:9092
KAFKA_BROKER_URL=kafka:9092
KAFKA_LISTENERS=LISTENER_INTERNAL://kafka:9092,LISTENER_EXTERNAL://localhost:29092
KAFKA_ADVERTISED_LISTENERS=LISTENER_INTERNAL://kafka:9092,LISTENER_EXTERNAL://localhost:29092
KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=LISTENER_INTERNAL:PLAINTEXT,LISTENER_EXTERNAL:PLAINTEXT
KAFKA_INTER_BROKER_LISTENER_NAME=LISTENER_INTERNAL
KAFKA_ADVERTISED_HOST_NAME=localhost
KAFKA_BROKER_ID=1
KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181
KAFKA_CREATE_TOPICS="user-login:1:1,processed-user-login:1:1,user-login-dlq:1:1"
ZOOKEEPER_CLIENT_PORT=2181
ZOOKEEPER_TICK_TIME=2000
ZOO_MY_ID=1
ZOO_PORT=2181
ZOO_SERVERS="server.1=zookeeper:2888:3888"
CONSUMER_GROUP=user-group
BOOTSTRAP_SERVERS=kafka:9092
ENABLE_AUTO_COMMIT=false
SOCKET_TIMEOUT=30000
SESSION_TIMEOUT=30000
AUTO_OFFSET_RESET=earliest
INPUT_TOPIC=user-login
OUTPUT_TOPIC=processed-user-login
DLQ_TOPIC=user-login-dlq
```

## Production Readiness

### 1. **Deployment to Kubernetes**
   - The solution can be deployed to **Kubernetes** using **Helm** for easy configuration management and scaling.
   - Kubernetes can handle the scaling of the consumer and producer services, ensuring that the system can handle large amounts of data in production.

### 2. **Monitoring with Prometheus, Grafana, and ELK**
   - **Prometheus** can be used for monitoring the health of Kafka, consumer, and producer services.
   - **Grafana** can visualize metrics from Prometheus for a real-time dashboard of system performance.
   - **ELK stack (Elasticsearch, Logstash, and Kibana)** can be used for centralized logging. Logs can be shipped to Logstash, processed, and stored in Elasticsearch, with Kibana used for visualization.

   You can add Prometheus, Grafana, and ELK by including the appropriate Docker containers or Helm charts in your Kubernetes setup.

### 3. **Kafka in Production**
   - **Replication**: In a production setup, Kafka topics should be configured with a replication factor greater than 1 to ensure high availability and fault tolerance.
   - **Partitioning**: Kafka topic partitioning helps distribute the load across multiple brokers, increasing scalability.

### 4. **Logging in Production**
   - Use **`LEVEL=INFO`** in `.env` for production to avoid verbose logging.
   - Set up log aggregation tools (such as **ELK** or **Prometheus logs**) to collect and visualize logs from all services in the pipeline.

## Troubleshooting Tips

If you encounter issues while running the project, here are some common problems and solutions:

### 1. **Kafka Consumer Not Receiving Messages**
   - **Cause**: The Kafka consumer may not be properly connected to the Kafka broker or may be misconfigured.
   - **Solution**:
     - Verify that the Kafka broker is running. You can check the logs of the Kafka container:
       ```bash
       docker logs pipeline-kafka
       ```
     - Ensure that the `KAFKA_BROKER_URL` environment variable is correctly set to the correct Kafka broker address in the `.env` file.
     - Check if the `user-login` topic exists. If not, create it using Kafka's CLI:
       ```bash
       kafka-topics.sh --create --topic user-login --partitions 1 --replication-factor 1 --bootstrap-server localhost:29092
       ```

### 2. **Messages Going to Dead Letter Queue (DLQ)**
   - **Cause**: The consumer may be rejecting valid messages due to incorrect validation logic.
   - **Solution**:
     - Review the validation logic in the consumer code to ensure that the fields (e.g., `UserID`, `AppVersion`, `DeviceType`) are being validated correctly.
     - Check the logs for any errors related to the DLQ. You can use `docker logs` to inspect the consumer logs:
       ```bash
       docker logs pipeline-consumer
       ```

### 3. **Kafka Connection Timeout**
   - **Cause**: Kafka might not be reachable from your consumer or producer.
   - **Solution**:
     - Check if the Kafka and Zookeeper containers are running properly by inspecting their logs:
       ```bash
       docker logs pipeline-kafka
       docker logs pipeline-zookeeper
       ```
     - Ensure that the `KAFKA_LISTENER` and `KAFKA_BROKER_URL` environment variables in the `.env` file are correctly configured for internal and external communication.

### 4. **Producer Not Sending Messages to Kafka**
   - **Cause**: The producer service may not be properly configured or may not be connecting to Kafka.
   - **Solution**:
     - Check the producer service logs to see if there are any connection issues or errors:
       ```bash
       docker logs pipeline-producer
       ```
     - Ensure that the `KAFKA_BROKER_URL` and `INPUT_TOPIC` are correctly set in the `.env` file for the producer.
     - Verify the Kafka broker is up and running by consuming from the topic directly:
       ```bash
       kafka-console-consumer --bootstrap-server localhost:29092 --topic user-login --from-beginning
       ```

### 5. **Service Not Starting or Exiting Unexpectedly**
   - **Cause**: There may be issues with the Docker containers or the environment variables.
   - **Solution**:
     - Check the Docker container logs to identify any errors during startup:
       ```bash
       docker logs <container_name>
       ```
     - Ensure that the `.env` file is properly configured and contains all the required environment variables.
     - Run `docker-compose down` followed by `docker-compose up --build` to rebuild the containers and clear any stale states.

### 6. **Topic Creation Fails**
   - **Cause**: Kafka may fail to create topics automatically if the configuration is incorrect or if permissions are not set correctly.
   - **Solution**:
     - Ensure that the `KAFKA_CREATE_TOPICS` environment variable in the `.env` file lists the correct topics and partition configurations.
     - Manually create the topics using Kafka's CLI:
       ```bash
       kafka-topics.sh --create --topic user-login --partitions 1 --replication-factor 1 --bootstrap-server localhost:29092
       kafka-topics.sh --create --topic processed-user-login --partitions 1 --replication-factor 1 --bootstrap-server localhost:29092
       kafka-topics.sh --create --topic user-login-dlq --partitions 1 --replication-factor 1 --bootstrap-server localhost:29092
       ```

### 7. **Kafka Logs Not Showing Consumer Activity**
   - **Cause**: The consumer might be configured to use a manual commit strategy, and the logs may not reflect offset commits.
   - **Solution**:
     - Ensure that the `ENABLE_AUTO_COMMIT` variable is set to `false` for manual offset control, and manually commit offsets in the code when processing is complete.
     - Check that the consumer group is correctly set in the `.env` file with the `CONSUMER_GROUP` variable.

### 8. **Graceful Shutdown Not Working**
   - **Cause**: The consumer may not be properly handling termination signals.
   - **Solution**:
     - Ensure that the shutdown logic is implemented correctly in the Go consumer to handle SIGINT and SIGTERM signals. Example code for graceful shutdown in Go:
       ```go
       sigs := make(chan os.Signal, 1)
       signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
       <-sigs
       // Cleanup and shutdown logic here
       ```

### 9. **Docker Compose Failing to Start Containers**
   - **Cause**: There may be conflicts with port bindings or missing dependencies.
   - **Solution**:
     - Ensure no other services are using the same ports as defined in your `docker-compose.yml` (e.g., `29092` for Kafka).
     - Use `docker-compose logs` to diagnose which service failed to start and why.

For additional support, please refer to the official Kafka documentation or open an issue on the GitHub repository.

## Conclusion

This solution provides a scalable, fault-tolerant real-time data pipeline using Kafka, Docker, and Go. The design ensures efficient message processing with a consumer that can handle retries and handle errors through the Dead Letter Queue. This setup can be easily deployed in production environments with Kubernetes and monitored using tools like Prometheus and Grafana.

For any questions or support, feel free to open an issue on the repository.

