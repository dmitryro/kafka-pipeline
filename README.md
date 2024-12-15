![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![License](https://img.shields.io/badge/license-MIT-blue)


## Table of Contents

* **[Introduction](#introduction)**
* **[Key Features](#key_features)**
* **[Design Choices](#design_choices)**
    * [Consumer Component Features](#consumer_component_features)
    * [Consumer Flow](#consumer_flow)
* **[Consumer Documentation](#consumer_documentation)**
    * [Consumer Overview](#consumer_documentation_overview)
    * [Consumer Design Choices](#consumer_documentation_design_choices)
    * [Consumer Component Features](#consumer_component_features)
    * [Consumer Flow](#consumer_flow)
    * [Consumer Environment Configuraton](#consumer_environment_configuration)
    * [Consumer Docker Configuraton](#consumer_docker_configuration)
    * [Consumer Logging and Monitoring](#consumer_logging_and_monitoring)
    * [Consumer Directory Laout](#consumer_documentation_directory_layout)
    * [Consumer Implementation](#consumer_implementation)
    * [Consumer Unit Tests](#consumer_documentation_unit_tests)
    * [Consumer Production Notes](#consumer_documentation_production_notes)

* **[Architecture Diagram](#archietcture_diagram)**

* **[Production Readiness](#production_readiness)**
    * [Common Steps](#producton_readiness_common_steps)
    * [Enhancements](#producton_readiness_enhancements)

* **[Project Deployment](#project_deployment)**
    * [Running the Project Locally](#running_locally)
    * [Production Deployment Steps](#production_deployment_steps)
    * [Deploying In The Cloud](#deploying_in_the_cloud)
    * [Deployment Commands](#deployment_commands) 
    * [Kubernetes and Container Orchestration](#production_kubernetes_orchestration)
    * [Logging and Alerging](#logging_and_alerging)


* **[Security and Compliance](#security_and_compliance)**
* **[Scalability](#scalability)**
* **[Scaling Strategies](#scaling_strategies)**
* **[Troubleshooting Tips](#troublesooting_tips)**
* **[Conclusion](#conclusion)**

## Introduction <a name="introduction"></a>

This project implements a real-time data streaming pipeline using **Apache Kafka**, **Docker**, and **Go**. It involves creating a system to consume, process, and produce data messages in Kafka topics, while ensuring scalability, fault tolerance, and efficient message handling. The pipeline consists of the following components:

1. **Kafka** for message ingestion and distribution.
2. **Docker** for containerization of services.
3. **Go-based consumer service** to process and produce data.
4. **Python-based producer service** for simulating data generation and pushing it to Kafka.

The solution involves setting up a Kafka consumer in Go that consumes messages from a Kafka topic (`user-login`), processes them based on certain rules, and publishes the processed data to another Kafka topic (`processed-user-login`). Any invalid messages are sent to a Dead Letter Queue (DLQ) topic (`user-login-dlq`). 

## Key Features <a name="key_features"></a>
- **Kafka consumer**: Reads messages from a Kafka topic (`user-login`), processes them based on certain checks, and publishes valid messages to the `processed-user-login` topic.
- **Dead Letter Queue (DLQ)**: Invalid messages are sent to the `user-login-dlq` topic.
- **Fault tolerance and retries**: The consumer ensures that messages are processed even in case of temporary issues, using retry logic with exponential backoff.
- **Graceful shutdown**: The application handles shutdown signals to close Kafka consumer and producer connections cleanly.

## Design Choices <a name="design_choices"></a>

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


## Consumer Documentation <a name="consumer_documentation"></a>

### Overview <a name="consumer_documentation_overview"></a>

The consumer component is designed to consume messages from a Kafka topic, validate and process those messages, and forward valid messages to another Kafka topic. It also handles invalid messages by placing them in a Dead Letter Queue (DLQ). This consumer is built to be highly robust, with error handling, retries, graceful shutdown, and filtering features.

### Design Choices <a name="consumer_documentation_design_choices"></a>
This consumer application is written in ```Go``` and leverages the ```confluentinc/confluent-kafka-go``` library for interacting with Apache Kafka. This choice offers several advantages:
   - **Go**: ```Go``` is a performant, statically typed language with excellent concurrency features, making it well-suited for building scalable and reliable message processing applications like this consumer.
   - **confluentinc/confluent-kafka-go**: This popular ```Go``` library provides a mature and user-friendly API for interacting with Kafka clusters. It offers features for consumer group management, message consumption, and producer functionality.



### Consumer Implementation <a name="consumer_implementation"></a>

#### ```main.go``` Breakdown <a name="consumer_documentation_main_go"></a>
The ```main.go``` file serves as the entry point for the consumer application. It defines various functions responsible for Kafka configuration, message processing, and graceful shutdown. Let's delve into each function's purpose, arguments, and return values.



#### Imports Overview
---
The following Go packages and external libraries are used in the application:

##### Standard Library Packages:
- **`context`**: Provides context management for cancellations, deadlines, and metadata across API boundaries.
- **`encoding/json`**: Enables encoding and decoding of JSON data.
- **`fmt`**: Used for formatted I/O operations.
- **`log`**: Provides logging capabilities for the application.
- **`net`**: Used for networking utilities like IP validation.
- **`os`**: Offers functionality for interacting with the operating system, such as environment variables and signals.
- **`os/signal`**: Facilitates handling of operating system signals.
- **`strings`**: Provides functions for string manipulation.
- **`sync`**: Offers concurrency primitives such as `WaitGroup`.
- **`syscall`**: Used for low-level system call handling.
- **`time`**: Handles time-based operations, such as delays and timestamps.

##### External Libraries:
- **`github.com/confluentinc/confluent-kafka-go/v2/kafka`**: Official Go client for Apache Kafka, used for producing and consuming messages.
- **`github.com/prometheus/client_golang/prometheus`**: Prometheus library for defining and managing custom metrics.
- **`github.com/prometheus/client_golang/prometheus/promhttp`**: Provides an HTTP handler for exposing Prometheus metrics.

##### Additional Libraries:
- **`net/http`**: Facilitates HTTP server implementation for metrics endpoint.

##### Note
- These imports enable essential functionalities, such as Kafka communication, Prometheus metrics tracking, HTTP server setup, and concurrent processing.
---



#### Consumer Data Types <a name="consumer_documentation_data_types"></a>
---
This section describes the data types (structs) used in the consumer application.


##### `Message`
```go
type Message struct {
    UserID     string `json:"user_id"` 
    AppVersion string `json:"app_version"`
    DeviceType string `json:"device_type"`
    IP         string `json:"ip"`
    Locale     string `json:"locale"` 
    DeviceID   string `json:"device_id"`
    Timestamp  int64  `json:"timestamp"`
}
```

**Description**:
This struct represents the structure of a raw message consumed from the Kafka input topic. It contains the necessary fields that are expected in the message.

**Fields**:
- `UserID` (type: `string`): The ID of the user associated with the message.
- `AppVersion` (type: `string`): The version of the application sending the message.
- `DeviceType` (type: `string`): The type of device used by the user.
- `IP` (type: `string`): The IP address of the device sending the message.
- `Locale` (type: `string`): The locale (language/region) of the user.
- `DeviceID` (type: `string`): The unique identifier of the device. 
- `Timestamp` (type: `int64`): The timestamp when the message was created. 

**Purpose**:
- This struct is used to unmarshal the raw JSON message received from Kafka.
- It serves as the base structure for validating and processing the message.

---

##### `ProcessedMessage`
```go
type ProcessedMessage struct {
    Message
    ProcessedAt string `json:"processed_at"`
}
```

**Description**:
This struct extends the `Message` struct and represents a processed message that includes a timestamp indicating when it was processed.

**Fields**:
- `Message` (type: `Message`): The original message, including all fields from the `Message` struct.
- `ProcessedAt` (type: `string`): The timestamp indicating when the message was processed, formatted in RFC3339 format.

**Purpose**:
- This struct is used to represent the message after it has been validated and processed, including a `ProcessedAt` timestamp.
- It is used for marshalling and publishing the processed message to the Kafka output topic.


#### Consumer Functions <a name="consumer_documentation_functions"></a>
---

This section provides a comprehensive overview of all functions implemented in `main.go`, including their purposes, input arguments, and returned values.

#### `publishWithRetry(producer *kafka.Producer, topic string, message []byte, retries int, delay time.Duration)`

**Description**:
This function attempts to publish a message to a Kafka topic. If the publishing fails, it retries the operation with a specified delay between attempts, up to a maximum number of retries.

**Input Arguments**:
- `producer`: The Kafka producer used to publish the message (type: `*kafka.Producer`).
- `topic`: The Kafka topic to publish the message to (type: `string`).
- `message`: The message to be published (type: `[]byte`).
- `retries`: The maximum number of retry attempts (type: `int`).
- `delay`: The delay duration between retry attempts (type: `time.Duration`).

**Returned Values**:
- Returns `nil` if the message is successfully published.
- Returns an `error` if all retries fail.

**Functionality**:
- Prepares the Kafka message with the specified topic and message payload.
- Publishes the message using the provided Kafka producer.
- If publishing fails, logs the error and retries the operation after waiting for the specified delay.
- Returns the last encountered error if all retry attempts are unsuccessful.

**Example Usage**:
```go
producer, err := kafka.NewProducer(&kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092",
})
if err != nil {
    log.Fatalf("Failed to create producer: %v", err)
}
defer producer.Close()

message := []byte("Hello, Kafka!")
topic := "example-topic"
retries := 5
delay := 2 * time.Second

err = publishWithRetry(producer, topic, message, retries, delay)
if err != nil {
    log.Fatalf("Failed to publish message: %v", err)
} else {
    log.Println("Message successfully published.")
}

```


#### `isPrivateIP(ip string) bool`

**Description**:  
This function checks if a given IP address is a private IP address.

**Input Arguments**:  
- `ip` (`string`): The IP address to check, represented as a string.

**Returned Values**:  
- Returns `true` if the provided IP address is private.  
- Returns `false` otherwise.

**Functionality**:  
1. Parses the input string to validate its format as an IP address.  
2. Checks if the IP address falls within the ranges defined for private IP addresses:  
   - `10.0.0.0` to `10.255.255.255` (Class A)  
   - `172.16.0.0` to `172.31.255.255` (Class B)  
   - `192.168.0.0` to `192.168.255.255` (Class C)  
3. Returns `true` if the IP matches any of the above ranges; otherwise, returns `false`.

**Example Usage**:  
```go
ip := "192.168.1.1"
if isPrivateIP(ip) {
    fmt.Printf("%s is a private IP address.\n", ip)
} else {
    fmt.Printf("%s is not a private IP address.\n", ip)
}

```


#### `isValidMessage(msg Message) bool`

**Description**:  
This function checks if a `Message` instance satisfies validation criteria to be considered valid.

**Input Arguments**:  
- `msg` (`Message`): The message object to validate, typically consisting of fields like `Key`, `Value`, and any additional metadata.

**Returned Values**:  
- Returns `true` if the message meets all validation criteria.  
- Returns `false` otherwise.

**Functionality**:  
1. Ensures the `Key` field of the message is non-empty, as it may be used to partition messages or maintain uniqueness.  
2. Validates that the `Value` field is non-empty to ensure the message contains meaningful content.  
3. Performs additional checks as necessary to confirm the message adheres to the application's specific requirements.

**Example Usage**:  
```go
msg := Message{
    Key:   "exampleKey",
    Value: []byte("exampleValue"),
}

if isValidMessage(msg) {
    fmt.Println("The message is valid.")
} else {
    fmt.Println("The message is invalid.")
}
```


##### `processMessage(message []byte) ([]byte, error)`

**Description**:  
Processes a Kafka message by performing necessary transformations.

**Input Arguments**:
- `message`: The Kafka message to process (type: `[]byte`).

**Returned Values**:
- `[]byte`: The processed message ready for publishing.
- `error`: Returns an error if the message processing fails.

**Functionality**:
- Parses the message and applies business logic transformations.
- Returns the modified message or an error if processing fails.

---


##### `processMessages(ctx context.Context, messageChan <-chan *kafka.Message, producer *kafka.Producer, outputTopic, dlqTopic string)`

**Description**:  
This function processes Kafka messages from the input channel, applies validation, and routes messages to the appropriate Kafka topic. It handles invalid messages by publishing them to a Dead Letter Queue (DLQ) topic.

**Input Arguments**:  
- `ctx`: A `context.Context` used to manage cancellation and deadlines for processing.  
- `messageChan`: A channel (`<-chan *kafka.Message`) from which incoming Kafka messages are received.  
- `producer`: A Kafka producer (`*kafka.Producer`) used for sending messages to Kafka topics.  
- `outputTopic`: The Kafka topic to which valid messages are published (`string`).  
- `dlqTopic`: The Kafka topic to which invalid messages are routed (`string`).  

**Returned Values**:  
- None (this function does not return any values).

**Functionality**:  
1. Listens to incoming Kafka messages from `messageChan`.  
2. For each message:
   - Validates the message using a validation function (e.g., `isValidMessage`).  
   - Publishes valid messages to the specified `outputTopic`.  
   - Routes invalid messages to the `dlqTopic`.  
3. Handles errors and ensures processing continues gracefully.  
4. Respects the context for graceful shutdown or cancellation.

**Example Usage**:  
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

messageChan := make(chan *kafka.Message)
producer := createProducer() // Assume createProducer initializes a Kafka producer.
outputTopic := "processed-messages"
dlqTopic := "dead-letter-queue"

go processMessages(ctx, messageChan, producer, outputTopic, dlqTopic)

```

---
##### `handleSignals(cancel context.CancelFunc, consumer *kafka.Consumer, producer *kafka.Producer)`

**Description**:  
This function listens for system signals to gracefully shut down the Kafka consumer, producer, and other application resources. It ensures proper cleanup during application termination.

**Input Arguments**:  
- `cancel`: A `context.CancelFunc` used to cancel any active contexts and signal shutdown.  
- `consumer`: The Kafka consumer (`*kafka.Consumer`) to be closed during shutdown.  
- `producer`: The Kafka producer (`*kafka.Producer`) to be closed during shutdown.  

**Returned Values**:  
- None (this function does not return any values).

**Functionality**:  
1. Waits for OS signals such as `SIGINT` or `SIGTERM` using a signal channel.  
2. Upon receiving a signal:
   - Calls `cancel` to terminate active contexts.  
   - Closes the Kafka consumer and producer to release resources.  
   - Logs the shutdown process for observability.  

**Example Usage**:  
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

consumer := createConsumer() // Assume createConsumer initializes a Kafka consumer.
producer := createProducer() // Assume createProducer initializes a Kafka producer.

go handleSignals(cancel, consumer, producer)

// Application logic here...
```
**Notes**:
- This function is designed to run concurrently, typically in a goroutine, to handle signals without blocking main application logic.
- Ensure proper error handling when closing the consumer and producer to manage edge cases.
- Always use this function to enable graceful shutdown in Kafka-based applications.

---

##### `startMetricsServer()`

**Description**:
Start Prometheus Metrics Server.

**Input Arguments**:
- None.

**Returned Values**:
- None.

**Functionality**:
- Start Prometheus Metrics Server.

**Example Usage**:
```go
startMetricsServer()
```

---

##### `init()`


**Description**:  
The `init` function initializes necessary configurations and resources before the program executes the `main` function. It is called automatically by Go during the program's initialization phase.

**Input Arguments**:  
- None (this is a Go standard initialization function and takes no arguments).

**Returned Values**:  
- None (this function does not return any values).

**Functionality**:  
1. Loads configuration parameters, such as environment variables, if required by the application.  
2. Initializes global variables or shared resources used throughout the application.  
3. Ensures that essential setup steps are completed before the program starts executing the main logic.  

**Example Usage**:  
This function is called automatically and does not need to be explicitly invoked in the code.  

```go
func init() {
    // Example: Set default configurations or environment variables
    log.Println("Initializing application configurations...")
    prometheus.MustRegister(kafkaMessagesProcessed)
}
---

##### `main()`

**Description**:  
The entry point of the application, orchestrating the setup and execution of the Kafka consumer and producer.

**Input Arguments**:
- None.

**Returned Values**:
- None.

**Functionality**:
- Reads environment variables for Kafka configurations.
- Sets up Kafka consumer and producer instances.
- Subscribes to the input topic and starts message processing.
- Handles graceful shutdown upon receiving termination signals.



### Consumer Component Features <a name="consumer_component_features"></a>

#### 1. **Message Validation and Filtering**
   - **Purpose**: Ensures that each message contains required fields and adheres to a predefined schema.
   - **Fields Checked**: 
     - `UserID`: The identifier for the user.
     - `AppVersion`: The version of the app generating the event.
     - `DeviceType`: The type of device the user is using (e.g., mobile, desktop).
   - **Action**: 
     - Valid messages are forwarded to the `processed-user-login` Kafka topic.
     - Invalid messages are placed into a Dead Letter Queue (DLQ) (`user-login-dlq`) for further inspection.

#### 2. **Error Handling and Retry Logic**
   - **Purpose**: Ensures that transient errors do not cause message loss.
   - **How It Works**: 
     - The consumer retries message processing up to a defined number of times if temporary errors occur (e.g., Kafka unavailability or validation issues).
     - If all retry attempts are exhausted, the message is sent to the DLQ.
     - Errors during message consumption or processing are logged for monitoring and debugging.

#### 3. **Graceful Shutdown**
   - **Purpose**: Ensures that the consumer can gracefully shut down, processing any in-flight messages before exiting.
   - **How It Works**: 
     - Upon receiving shutdown signals (`SIGINT`, `SIGTERM`), the consumer stops consuming new messages and finishes processing the current batch.
     - Logs and metrics are flushed before the consumer stops.

#### 4. **Backpressure Handling**
   - **Purpose**: Prevents the system from being overwhelmed by too many messages.
   - **How It Works**: 
     - The consumer implements rate-limiting and backpressure handling by controlling the rate at which messages are consumed and processed.
     - This helps manage high message throughput and ensures the system does not exceed capacity.

#### 5. **Logging and Metrics**
   - **Purpose**: Tracks and logs the consumer’s activity for monitoring and troubleshooting.
   - **Log Types**: 
     - **Success Logs**: Log entries for successfully processed messages.
     - **Error Logs**: Log entries for message validation failures and retry attempts.
     - **Retry Logs**: When a message is retried due to transient issues.
   - **Metrics**:
     - **Message Consumption Rate**: Tracks how fast messages are being consumed.
     - **Retry Count**: Number of times a message has been retried.
     - **DLQ Count**: Number of messages that have been sent to the Dead Letter Queue.
     - **Message Processing Time**: The time taken to process each message.
   - **Integration**: The logs can be forwarded to centralized logging systems like Datadog or ELK for detailed monitoring.

#### 6. **Dead Letter Queue (DLQ)**
   - **Purpose**: Holds messages that cannot be processed due to validation errors or failures after retry attempts.
   - **How It Works**: 
     - If a message fails validation or processing after the retry limit, it is sent to a Kafka topic (`user-login-dlq`).
     - This ensures that no data is lost and can be inspected manually or reprocessed later.

#### 7. **Kafka Consumer Group**
   - **Purpose**: Allows multiple consumer instances to share the load of consuming messages from Kafka.
   - **How It Works**: The consumer is part of a Kafka consumer group that distributes partitions across all instances of the consumer, ensuring load balancing and fault tolerance.

#### 8. **Scalability**
   - **Purpose**: Allows the consumer to scale horizontally to handle increased load.
   - **How It Works**: 
     - To scale the consumer, you can increase the number of consumer instances in the same Kafka consumer group.
     - Kafka automatically balances the load by distributing topic partitions across the available consumers.


### Consumer Flow <a name="consumer_flow"></a>

1. **Consume Message**: The consumer listens to the `user-login` Kafka topic.
2. **Message Validation and Filtering**: Each message is validated for required fields (`UserID`, `AppVersion`, `DeviceType`). Invalid messages are forwarded to the DLQ, while valid messages are processed further.
3. **Retry Logic**: If a transient failure occurs (e.g., network issues), the message will be retried a predefined number of times.
4. **Forward Valid Message**: Valid messages are forwarded to the `processed-user-login` topic.
5. **Graceful Shutdown**: Upon receiving a shutdown signal, the consumer gracefully finishes processing messages and exits.

### Consumer Environment Configuration <a name="consumer_environment_configuration"></a>

The consumer is configured via the `.env` file and can be customized with the following parameters:

- `KAFKA_BROKER_URL`: The address of the Kafka broker (e.g., `localhost:29092`).
- `INPUT_TOPIC`: The Kafka topic to consume messages from (`user-login`).
- `OUTPUT_TOPIC`: The Kafka topic to send valid, processed messages to (`processed-user-login`).
- `DLQ_TOPIC`: The topic for invalid messages (`user-login-dlq`).
- `CONSUMER_GROUP`: The name of the consumer group (used for consumer group management in Kafka).
- `RETRY_LIMIT`: The maximum number of retry attempts for a message before it is sent to the DLQ.
- `LOG_LEVEL`: The logging level (e.g., `debug`, `info`, `warn`, `error`).

### Consumer Docker Configuration <a name="consumer_docker_configuration"></a>

The consumer is containerized using Docker. Below is the `Dockerfile` and `docker-compose.yml` used for the consumer service.

#### Dockerfile

```Dockerfile
# Use the latest  Go image
FROM golang:1.23.4

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the application code
COPY . .

# Build the Go application
RUN go build -o data-consumer main.go

# Run the Go application
CMD ["./data-consumer"]
```

### docker-compose.yml 
```docker-compose.yml

services:
  data-consumer:
    container_name: ${PROJECT_NAME}-consumer
    env_file:
      - .env
    build:
      context: ./data-consumer
    depends_on:
      - kafka
    networks:
      - kafka-network
```


### Consumer Logging and Monitoring <a name="consumer_logging_and_monitoring"></a>


### Consumer Directory Layout <a name="consumer_documentation_directory_layout"></a>
The consumer logic resides within the data-consumer directory. Here's a breakdown of its contents:

```
data-consumer/
├── Dockerfile          (Docker build configuration)
├── go.mod              (Go module dependency file)
├── go.sum               (Checksum file for dependencies)
├── main_test.go         (Unit Tests to the functions in main.go)
└── main.go              (Go source code for the consumer application)
```

### Consumer Unit Tests <a name="consumer_documentation_unit_tests"></a>

This section provides a detailed explanation of the unit tests included in the `main_test.go` file. Each test validates specific functionality within the consumer application to ensure correctness, reliability, and fault tolerance.

#### `TestIsValidMessage`

**Description**:
Tests the `isValidMessage` function, which checks the validity of incoming Kafka messages.

**Test Cases**:
1. **Valid Message**: Verifies that a properly formatted message with all required fields is deemed valid.
2. **Missing Required Field**: Tests a message missing the `user_id` field and expects it to be invalid.
3. **Missing Timestamp Field**: Validates that messages without a `timestamp` field are considered invalid.
4. **Invalid JSON Format**: Ensures that malformed JSON messages are flagged as invalid.
5. **Incorrect Field Type**: Checks that a message with a non-numeric `timestamp` field is invalid.

---

#### `TestProcessMessage`

**Description**:
Tests the `processMessage` function, which processes valid Kafka messages.

**Test Cases**:
1. **Valid Message**: Confirms that a properly formatted message is processed correctly and returns expected structured data.
2. **Invalid Message**: Ensures that messages missing required fields are flagged as invalid and not processed.

---

#### `TestPublishWithRetry`

**Description**:
Validates the `publishWithRetry` function, which publishes messages to Kafka with retry logic for fault tolerance.

**Test Cases**:
1. **Retry Logic**: Simulates an initial failure in publishing, followed by a successful retry, and ensures the retry logic works correctly.
2. **Producer Call Validation**: Verifies that the mock Kafka producer is called the expected number of times during retries.

---

#### `TestIsPrivateIP`

**Description**:
Tests the `isPrivateIP` function, which checks if a given IP address belongs to a private range.

**Test Cases**:
1. **Private IP**: Validates that a typical private IP address (e.g., `192.168.1.1`) is correctly identified.
2. **Public IP**: Ensures that a public IP address (e.g., `8.8.8.8`) is not marked as private.
3. **Loopback IP**: Confirms that loopback addresses (e.g., `127.0.0.1`) are correctly flagged.
4. **Reserved IP**: Validates detection of reserved ranges, such as APIPA (`169.254.1.1`), as private.

---

#### `TestClose`

**Description**:
Tests the `Close` method of the Kafka producer to ensure resources are released gracefully.

**Test Cases**:
1. **Successful Close**: Mocks the producer's `Close` method and verifies that it completes without errors.
2. **Method Invocation**: Ensures the `Close` method is invoked as expected.

---

#### `TestHandleError`

**Description**:
Tests the `handleError` function, which handles application errors (e.g., logging or custom error handling).

**Test Cases**:
1. **Error Handling**: Validates that the function does not panic when called with an error.
2. **Side Effects**: Ensures any side effects (e.g., logging) occur as expected.

---

#### Mock Implementation

The tests utilize a `MockProducer` to simulate interactions with Kafka producers. This mock implementation validates producer behavior without requiring actual Kafka connections. Key methods include:
- `Produce`: Simulates message production, with customizable return values for testing success and failure scenarios.
- `Close`: Simulates closing the producer and ensures proper invocation during shutdown.

---

#### Dependencies

The tests use the following libraries:
- **Testify**: For assertions and mocking.
- **Go Testing Package**: Provides the standard testing framework.

Each test case ensures the application behaves as expected under various scenarios, contributing to the overall reliability and robustness of the system.




### Consumer Producton Notes<a name="consumer_documentation_production_notes"></a>



## Architecture Diagram <a name="archietcture_diagram"></a>

![Real-time Streaming Data Pipeline Architecture](./images/architecture_diagram.png)



### Running the Project Locally <a name="running_locally"></a>

#### Prerequisites
- **Docker** and **Docker Compose** must be installed. 
- **Kafka** and **Zookeeper** will be run as Docker containers.

#### Steps to Run the Project

1. Clone this repository:
   ```bash
   git clone git@github.com:dmitryro/kafka-pipeline.git data_pipeline 
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


#### Environment Variables

- **LEVEL**: Controls the logging level. Set to `DEBUG` in `.env` for development and `INFO` for production.
- **KAFKA_LISTENER**: The Kafka broker URL for internal communication (e.g., `kafka:9092`).
- **KAFKA_BROKER_URL**: The Kafka broker URL for external communication (e.g., `localhost:29092`).
- **KAFKA_CREATE_TOPICS**: Comma-separated list of Kafka topics to create on startup.
- **KAFKA_ZOOKEEPER_CONNECT**: Connection string for Zookeeper.

See `.env` for all available environment variables and their descriptions.

#### Sample .env File

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

## Production Readiness <a name="production_readiness"></a>

### Producton Readiness Common Steps  <a name="production_readiness_common_steps"></a>
#### 1. **Deployment to Kubernetes**
   - The solution can be deployed to **Kubernetes** for managing and scaling services in production. Kubernetes helps in automating the deployment, scaling, and management of containerized applications.
   - **Helm** can be used for easy configuration management and deployment of the system. Helm charts simplify the deployment process by packaging Kubernetes resources like deployments, services, and persistent volumes into reusable templates.
   - **Deployment on AWS EKS (Elastic Kubernetes Service)** or other managed Kubernetes services is recommended for better scalability and ease of maintenance. EKS provides a managed Kubernetes environment that can be scaled as needed, with built-in security, monitoring, and high availability.
   - The system should include horizontal scaling for both the Kafka producer and consumer services, ensuring that the pipeline can handle a growing volume of data without downtime.

#### 2. **Monitoring and Logging in Production**
   - For monitoring, integrate with **Prometheus** and **Grafana** to track the health of Kafka, consumer, and producer services.
   - **Prometheus** will gather metrics, while **Grafana** can be used to create dashboards for real-time monitoring.
   - Use the **ELK stack (Elasticsearch, Logstash, Kibana)** for centralized logging. Logs from all services, including Kafka brokers, producers, and consumers, can be aggregated in Elasticsearch, and visualized in Kibana for troubleshooting and performance monitoring.

#### 3. **Kafka in Production**
   - **Replication**: Kafka topics should have a replication factor greater than 1 for high availability. This ensures that Kafka data remains available even if a broker fails.
   - **Partitioning**: Kafka topic partitioning should be configured according to throughput requirements. More partitions allow better distribution of the data across multiple Kafka brokers, improving scalability.
   - **Kafka Connect** can be used for integrating external systems (such as databases or third-party APIs) to produce or consume data from Kafka topics.

#### 4. **Scaling Considerations**
   - The Kafka cluster should be scaled horizontally by adding more brokers to the Kafka cluster as needed.
   - The consumer application should also be scaled horizontally by adding more pods or containers. Each consumer should be part of a consumer group to ensure that messages are processed in parallel across multiple instances.

#### 5. **Automated Deployments and CI/CD**
   - Implement a **CI/CD pipeline** using **GitLab CI**, **Jenkins**, or **GitHub Actions** to automate the testing, building, and deployment of the system to Kubernetes.
   - The pipeline should include steps to:
     - Build Docker images for the producer and consumer services.
     - Push the Docker images to a container registry (e.g., Docker Hub, Amazon ECR).
     - Deploy the services to Kubernetes (e.g., using Helm charts).
     - Monitor health and automatically scale services based on resource utilization or incoming data volume.

#### 6. **Security and Compliance**
   - Implement **role-based access control (RBAC)** in Kubernetes to ensure that only authorized users and services can access Kafka topics or deploy updates to the pipeline.
   - **Audit logging** for all Kafka interactions can help with security and compliance, especially in regulated industries.
   - **TLS encryption** :Ensure secure communication with Kafka brokers by enabling **SSL/TLS** encryption for both producers and consumers.
   - Use **IAM roles** for secure access to cloud services like S3 or MSK, ensuring least privilege access.

#### 7. **Fault Tolerance and High Availability**
   - Ensure that **Kafka brokers** are deployed in a fault-tolerant configuration with replication across multiple availability zones to avoid data loss in case of broker failure.
   - Kafka consumers and producers should be deployed in a manner that ensures high availability, possibly using multiple instances across different Kubernetes pods or nodes.
   - Implement **Health Checks** for Kafka brokers, producers, and consumers to monitor their availability and restart them automatically in case of failure.
   - Consider implementing **circuit breaker** patterns in case of failures in external systems.
   - Design the application to handle Kafka broker failures and allow for graceful recovery.

#### 8. **Backup and Disaster Recovery**
   - **Kafka Backups**: Implement a backup strategy for Kafka logs and topic data. Periodic snapshots of the Kafka data can be taken to ensure recovery in case of catastrophic failure.
   - **Disaster Recovery Plan**: In the event of a disaster, ensure that backup data can be restored to a new Kafka cluster quickly, minimizing downtime and data loss.

#### 9. **Cost Optimization**
   - Use **auto-scaling** in Kubernetes to adjust the number of producer and consumer pods based on workload, ensuring that the system can scale up during high data traffic and scale down during idle times.
   - Optimize the **Kafka cluster's storage** by adjusting the retention period of topics and using **log compaction** for certain topics to save disk space.
   - Monitor and adjust **instance types and resource allocation** for Kafka brokers and consumer services to avoid over-provisioning while ensuring adequate performance.

#### 10. **Error Handling and Alerts**:
   - Implement comprehensive error handling to gracefully handle failures in Kafka message processing.
   - Set up alerts using tools like **Prometheus Alertmanager** or **Datadog** to monitor for issues like consumer lag, application crashes, and resource utilization.


## Project Deployment <a name="project_deployment"></a>


### Production Deployment Steps <a name="production_deployment_steps"></a>

To deploy this application in production, follow these steps:

   1. **Containerization:**
   - Use Docker to package the application with all dependencies.
   - Create separate Dockerfiles for development and production environments to optimize builds.

   2. **Orchestration:**
   - Use Kubernetes to manage containers at scale.
   - Define the following Kubernetes resources:
     - **Deployment:** For Kafka consumer and producer services with rolling updates.
     - **ConfigMaps:** To store application configurations like topic names and log levels.
     - **Secrets:** To securely store sensitive information like Kafka credentials.
     - **Services:** For internal communication between components.
     - **Ingress:** To expose application endpoints securely (if needed).

   3. **Infrastructure:**
   - Choose a reliable cloud provider (e.g., AWS, GCP, Azure).
   - Use managed Kafka services like Confluent Cloud or AWS MSK to reduce operational overhead.
   - Ensure a robust load balancer (e.g., Kubernetes Ingress or AWS ALB) for high availability.

   4. **Monitoring and Logging:**
   - Integrate **Prometheus** for metrics collection and **Grafana** for visualization.
   - Set up centralized logging with the **ELK Stack** (Elasticsearch, Logstash, Kibana).
   - Implement alerting tools like **PagerDuty** or **Opsgenie** for incident management.

   5. **Security:**
   - Enable Kafka encryption (SSL/TLS) and authentication (SASL).
   - Use network policies in Kubernetes to restrict pod communication.
   - Regularly audit Kafka ACLs to ensure least privilege access.

   6. **Disaster Recovery:**
   - Enable multi-zone replication for Kafka brokers.
   - Automate backups for Kafka data and configuration files.
   - Perform periodic recovery drills to validate backup integrity.


### Deploying In The Cloud <a name="deploying_in_the_cloud"></a>


### Deployment Commands<a name="deployment_commands"></a>
   1. **Build Docker Images:**
   ```bash
   docker build -t kafka-consumer:latest ./consumer
   docker build -t kafka-producer:latest ./producer
   ```

   2. **Run Locally with Docker Compose:**
   ```bash
   docker-compose up
   ```

   3. **Deploy to Kubernetes:**
   ```bash
   kubectl apply -f k8s/deployment.yaml
   kubectl apply -f k8s/service.yaml
   kubectl apply -f k8s/configmap.yaml
   kubectl apply -f k8s/secret.yaml
   ```

   4. **Monitor Deployment:**
   ```bash
   kubectl get pods
   kubectl logs -f <pod-name>
   ```

### Kubernetes and Container Orchestration <a name="production_kubernetes_orchestration"></a>

### Logging and Alerging <a name="logging_and_alerging"></a>

### Production Readiness Enhancements <a name="production_readiness_enhancements"></a>

To ensure the application is production-ready, consider adding the following components:

#### 1. **Monitoring and Observability:**
   - Use **Prometheus** for system metrics.
   - Implement **Grafana** dashboards for real-time visualization.
   - Set up a centralized logging stack (e.g., **ELK**, **Fluentd**, or **Loki**).

#### 2. **Security:**
   - Enable **Kafka encryption** using SSL/TLS.
   - Configure SASL for authentication (e.g., SASL-PLAIN, SASL-SCRAM).
   - Use RBAC in Kubernetes for access control and enforce network policies.

#### 3. **Scalability:**
   - Leverage **Kafka Streams** or a framework like **Apache Flink** for complex data processing.
   - Use horizontal pod autoscaling in Kubernetes based on CPU/memory utilization.
   - Optimize Kafka retention policies and partition configurations.

#### 4. **CI/CD Pipeline:**
   - Implement a pipeline using tools like **GitHub Actions**, **CircleCI**, or **Jenkins**.
   - Automate Docker image builds, Kubernetes deployments, and smoke tests.

#### 5. **Testing:**
   - Perform end-to-end testing to validate Kafka message flow.
   - Simulate high-throughput scenarios using tools like **kafka-producer-perf-test.sh**.
   - Monitor consumer lag during stress testing.

#### 6. **Data Governance:**
   - Enable schema validation using **Confluent Schema Registry**.
   - Implement data versioning to handle backward and forward compatibility.


## Security and Compliance <a name="security_and_compliance"></a>
#### **IAM Roles for Kafka**

Cloud-based Kafka services like Amazon MSK (Managed Streaming for Apache Kafka) and Confluent Cloud rely on Identity and Access Management (IAM) roles for securing access to Kafka resources. IAM roles are used to authenticate and authorize clients, services, and applications interacting with Kafka clusters.

##### **Amazon MSK IAM Roles**

In MSK, IAM roles are used to control access to your Kafka brokers and Kafka data within AWS. You can use IAM roles to:

- **Grant permissions to clients:** Through the use of IAM policies, you can control which users, roles, or services can produce, consume, or administer Kafka topics.
- **Authenticate clients:** MSK supports **IAM authentication** for producers and consumers to securely connect to Kafka brokers. IAM roles can be assigned to EC2 instances or services like AWS Lambda to authenticate without using traditional usernames and passwords.
- **Access Control:** Policies can be attached to IAM roles, controlling access based on Kafka resources like topics and consumer groups.

IAM roles for MSK are managed through AWS Identity and Access Management (IAM), and the appropriate permissions must be granted to allow the Kafka client applications to interact with MSK clusters.

##### Example: IAM Policy for MSK Consumer

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "kafka:DescribeCluster",
                "kafka:DescribeTopic",
                "kafka:ListTopics",
                "kafka:GetRecords",
                "kafka:Consume"
            ],
            "Resource": "arn:aws:kafka:region:account-id:cluster/cluster-name/*"
        }
    ]
}
```

#### Confluent Cloud IAM Roles

Confluent Cloud provides a robust IAM (Identity and Access Management) system to control access to Kafka resources. It integrates with cloud-native IAM systems like AWS IAM, Google Cloud IAM, and Azure AD to enable seamless and secure access control. With Confluent Cloud, you can define fine-grained permissions for managing Kafka clusters, topics, consumer groups, and other resources.

##### **Key IAM Roles in Confluent Cloud**

- **Administrator**: Full access to all resources and configurations within the Confluent Cloud environment. This role can manage Kafka clusters, create and delete topics, and manage IAM policies.
  
- **Kafka Cluster Admin**: Can create and manage Kafka clusters, configure brokers, and manage topic configurations. However, they do not have access to non-Kafka services like connectors, schemas, or user management.
  
- **Developer**: Can produce and consume messages to/from Kafka topics and create topics, but has limited access to administrative functionalities. Developers typically focus on managing their specific applications.
  
- **Viewer**: Can only view the configuration of Kafka resources, including topic details, consumer groups, and cluster configurations. This role does not allow any changes or access to message data.
  
- **Schema Registry Admin**: Can manage schemas within the Schema Registry but does not have access to Kafka cluster or other non-schema resources.

##### **Assigning IAM Roles in Confluent Cloud**

IAM roles are assigned at different levels, including:

- **Organization level**: Users can be assigned roles that give access to all resources within the Confluent Cloud organization.
- **Cluster level**: Roles can be restricted to a specific Kafka cluster or specific topics within that cluster.
- **Topic level**: Fine-grained access can be applied, such as allowing a user to only produce messages to a specific topic.

Roles are assigned through the Confluent Cloud UI or via the API by the administrator.

##### **Best Practices for IAM Role Management in Confluent Cloud**

- **Principle of Least Privilege**: Always assign the least amount of privilege necessary to perform the required tasks. For example, a developer should not be granted administrator permissions unless absolutely necessary.
- **Use Role-based Access Control (RBAC)**: RBAC allows administrators to define roles with specific permissions for different users or services within the organization.
- **Monitor Role Assignments**: Regularly review and audit IAM roles to ensure that only authorized users and services have access to sensitive Kafka resources.
- **Use Multi-Factor Authentication (MFA)**: Enhance security by enabling MFA for users with elevated IAM roles, such as administrators.

## Scalability <a name="scalability"></a>
As the dataset grows, the application should be designed to scale efficiently. Here are the key strategies for scaling:

1. **Horizontal Scaling of Consumers**:
   - You can scale the number of Kafka consumers to handle increased traffic. Kafka allows multiple consumers to read from the same topic by creating multiple consumer instances in different processes or containers. This ensures that the workload is distributed evenly.
   - Use a load balancer or Kubernetes to manage consumer scaling automatically based on CPU or memory usage.

2. **Kafka Partitioning**:
   - To improve throughput and distribute data processing more evenly, increase the number of partitions for Kafka topics. This allows consumers to read from different partitions in parallel, enhancing the throughput of the system.

3. **Backpressure Handling**:
   - In case of increased load, implement backpressure handling techniques, such as controlling the rate at which data is processed or batching the messages, to avoid overwhelming the system.

4. **Database Scaling**:
   - If the processed data is being stored in a database, ensure that the database can handle the increasing load. This may involve database sharding, read replicas, or using distributed databases that can scale horizontally.

5. **Cloud Resources**:
   - If using cloud services like AWS, GCP, or Azure, ensure auto-scaling is enabled for Kafka brokers and application instances. This ensures that the infrastructure adapts to growing loads without manual intervention.


## Scaling Strategies <a name="scaling_strategies"></a>

As the dataset grows, this application can scale effectively with the following strategies:

   1. **Kafka Partitioning:**
   - Increase the number of partitions in Kafka topics to allow parallel processing.
   - Use a key-based partition strategy to ensure data consistency and load balancing.

   2. **Consumer Scaling:**
   - Add more consumers in the same consumer group to scale horizontally.
   - Monitor consumer lag using tools like **Kafka Lag Exporter** to identify bottlenecks.

   3. **Optimizing Kafka Configuration:**
   - Tune Kafka settings like `retention.ms` and `segment.bytes` to handle large datasets efficiently.
   - Configure `min.insync.replicas` to ensure data durability while maintaining performance.

   4. **Resource Scaling:**
   - Use Kubernetes **Horizontal Pod Autoscaler** to add or remove pods based on CPU/memory utilization.
   - Scale Kafka brokers vertically (adding more resources) or horizontally (adding more brokers).

   5. **Enhanced Processing:**
   - Use frameworks like **Kafka Streams** or **Apache Flink** for stateful processing.
   - Consider data batch processing for non-real-time use cases with tools like **Apache Spark**.

## Troubleshooting Tips <a name="troublesooting_tips"></a>

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

## Conclusion <a name="conclusion"></a> 
This solution provides a scalable, fault-tolerant real-time data pipeline using Kafka, Docker, and Go. The design ensures efficient message processing with a consumer that can handle retries and handle errors through the Dead Letter Queue. This setup can be easily deployed in production environments with Kubernetes and monitored using tools like Prometheus and Grafana.

For any questions or support, feel free to open an issue on the repository.
