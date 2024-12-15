package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

/**
 * ProducerInterface defines the methods required for a Kafka producer.
 * Implementations of this interface should handle sending messages
 * to a Kafka topic, managing delivery reports, and ensuring fault tolerance.
 * Using this interface should make the code better decoupled and simpler to unit test.
 */
type ProducerInterface interface {
	Produce(*kafka.Message, chan kafka.Event) error
	Close() error // Ensure Close() returns an error
}

/**
 * ConsumerInterface defines the methods required for a Kafka consumer.
 * Implementations of this interface should handle subscribing to Kafka topics,
 * consuming messages, and processing them in a fault-tolerant and scalable manner.
 * Using this interface should make the code better decoupled and simpler to unit test.
 */
type ConsumerInterface interface {
	Subscribe(topic string, cb kafka.RebalanceCb) error
	Poll(timeoutMs int) kafka.Event
	Close() error
}

/**
 * Message:
 *
 * Represents a message received from the Kafka topic.
 *
 * Fields:
 *   - UserID: User ID associated with the message.
 *   - AppVersion: App version used to generate the message.
 *   - DeviceType: Type of device used to generate the message.
 *   - IP: IP address of the device.
 *   - Locale: Locale of the device.
 *   - DeviceID: Unique identifier of the device.
 *   - Timestamp: Timestamp of the message generation.
 */
type Message struct {
	UserID     string `json:"user_id"`
	AppVersion string `json:"app_version"`
	DeviceType string `json:"device_type"`
	IP         string `json:"ip"`
	Locale     string `json:"locale"`
	DeviceID   string `json:"device_id"`
	Timestamp  int64  `json:"timestamp"`
}

/**
 * ProcessedMessage:
 *
 * Represents a processed message, including the original message and a timestamp of processing.
 *
 * Fields:
 *   - Message: The original message received from the Kafka topic.
 *   - ProcessedAt: Timestamp indicating when the message was processed.
 */
type ProcessedMessage struct {
	Message
	ProcessedAt string `json:"processed_at"`
}

/**
 * kafkaMessagesProcessed is a Prometheus CounterVec metric that tracks the total number of Kafka messages processed.
 * 
 * This metric includes a "result" label, which can be used to categorize messages
 * by their processing outcome (e.g., "success", "failure").
 *
 * Usage:
 * - Increment the counter with a specific result label after processing each message.
 * - Register this metric with the Prometheus registry during application initialization.
 * - Use Prometheus scrapers or monitoring dashboards to visualize and alert on this metric.
 *
 * Example:
 * kafkaMessagesProcessed.WithLabelValues("success").Inc()
 * kafkaMessagesProcessed.WithLabelValues("failure").Inc()
 */
var (
	kafkaMessagesProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_processed_total",
			Help: "Total number of Kafka messages processed.",
		},
		[]string{"result"},
	)
)

/**
 * KafkaProducerWrapper
 *
 * Represents a wrapper around Producer object to allow more efficient contract and decoupling.
 *
 * Fields:
 *   - *kafka.Producer - the producer to be wrapped for decoupling
 */
type KafkaProducerWrapper struct {
	*kafka.Producer
}

/**
 * Close gracefully shuts down the KafkaProducerWrapper by closing the underlying Kafka producer.
 * 
 * This method ensures that any resources held by the Kafka producer are properly released, 
 * preventing resource leaks or hanging connections. It is typically called during the 
 * application's shutdown process to clean up.
 *
 * @param: None 
 *
 * @return:
 * - An error (if any) encountered while closing the producer. In this implementation, 
 *   it always returns nil, as the underlying producer's `Close` method does not produce an error.
 *
 * Example:
 * defer producerWrapper.Close()
 */
func (k *KafkaProducerWrapper) Close() error {
	k.Producer.Close()
	return nil
}

/**
 * init initializes the application-wide Prometheus metrics and registers them with the global
 * Prometheus metrics registry.
 * 
 * The `init` function is a special function in Go that is automatically executed once when the 
 * package is imported. It is commonly used for initialization tasks such as setting up metrics, 
 * logging configurations, or global state.
 * 
 * This `init` function registers the custom Prometheus metrics defined in the application, 
 * ensuring they are available for collection and exposure to monitoring systems.
 *
 * @param: None 
 *
 * @return: void
 * - This function does not return any value. 
 *
 * Usage:
 * No explicit invocation is needed as Go calls `init` automatically during the package initialization.
 */
func init() {
    // Register Prometheus metrics
	prometheus.MustRegister(kafkaMessagesProcessed)
}

/**
 * main:
 *
 * The entry point of the application that sets up the Kafka consumer and producer, configures signal handling 
 * for graceful shutdown, initializes a worker pool for concurrent message processing, and starts the main 
 * Kafka consumer loop.
 *
 * Detailed Workflow:
 * 1. Environment Variable Validation:
 *    - Reads the required Kafka topic names (input, output, and dead-letter queue) from environment variables.
 *    - Ensures all necessary environment variables are set, logging an error and terminating the program otherwise.
 *
 * 2. Kafka Consumer and Producer Initialization:
 *    - Creates a Kafka consumer configured to connect to the Kafka cluster and subscribe to the input topic.
 *    - Creates a Kafka producer to send processed messages to the output and dead-letter queue (DLQ) topics.
 *    - Both are managed using the Confluent Kafka Go client library.
 *    - If any initialization fails, logs the error and exits the application.
 *
 * 3. Producer Wrapper:
 *    - Wraps the Kafka producer in a custom `KafkaProducerWrapper` to simplify closing the producer and enable
 *      its use with a consistent interface (`ProducerInterface`) throughout the application.
 *
 * 4. Signal Handling:
 *    - A separate goroutine handles OS signals (e.g., SIGINT) for graceful shutdown of the consumer, producer,
 *      and worker pool.
 *
 * 5. Worker Pool:
 *    - Launches a fixed number of worker goroutines to process incoming Kafka messages concurrently.
 *    - Workers receive messages from a channel and handle processing and publishing results to output or DLQ topics.
 *
 * 6. Main Kafka Consumer Loop:
 *    - Fetches messages from the input topic using the consumer's `Poll` method.
 *    - For each received message:
 *        a. Logs the message.
 *        b. Sends the message to the processing channel for the worker pool.
 *    - Handles any Kafka errors reported during polling.
 *    - Stops consuming and processing messages gracefully when a shutdown signal is received.
 *
 * @param: None 
 *
 * @return: void
 * - This function does not return any value.
 *
 * Usage:
 * - This function does not require explicit invocation; it runs automatically when the application starts.
 * - Ensure all required environment variables are set before running the application.
 */
func main() {

    // Initialize the Kafka consumer and producer
	inputTopic := os.Getenv("KAFKA_INPUT_TOPIC")
	outputTopic := os.Getenv("KAFKA_OUTPUT_TOPIC")
	dlqTopic := os.Getenv("KAFKA_DLQ_TOPIC")

	if inputTopic == "" || outputTopic == "" || dlqTopic == "" {
		log.Fatal("One or more required environment variables are not set.")
	}

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BOOTSTRAP_SERVERS"),
		"group.id":          os.Getenv("KAFKA_CONSUMER_GROUP"),
		"auto.offset.reset": os.Getenv("KAFKA_AUTO_OFFSET_RESET"),
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %s", err)
	}
	defer consumer.Close()

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BOOTSTRAP_SERVERS"),
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}
	defer producer.Close()

	// Wrap the producer
	producerWrapper := &KafkaProducerWrapper{Producer: producer}

	// Continue using the wrapped producer
	err = consumer.Subscribe(inputTopic, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topic: %s", err)
	}

	// Set up for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	messageChan := make(chan *kafka.Message, 100)

	go handleSignals(cancel, consumer, producerWrapper) // Pass wrapped producer as ProducerInterface

	// Worker pool to process messages concurrently
	workerPoolSize := 10
	wg.Add(workerPoolSize)
	for i := 0; i < workerPoolSize; i++ {
		go func() {
			defer wg.Done()
			processMessages(ctx, messageChan, producerWrapper, outputTopic, dlqTopic)
		}()
	}

	log.Println("Consumer is running...")

	// Main polling loop to fetch messages from the Kafka topic
	for {
		select {
		case <-ctx.Done():
			close(messageChan)
			wg.Wait()
			return
		default:
			event := consumer.Poll(100)
			if event == nil {
				continue
			}

			switch ev := event.(type) {
			case *kafka.Message:
				log.Printf("Received message: %s", string(ev.Value)) // Log the message
				messageChan <- ev
			case kafka.Error:
				log.Printf("Consumer error: %v", ev)
			}
		}
	}
}

/**
 * processMessages:
 *
 * Processes messages from the message channel, validates them, and publishes them to either the output topic 
 * (for valid messages) or the Dead Letter Queue (DLQ) topic (for invalid messages). Implements retry logic for 
 * publishing failures and gracefully exits when the context is canceled.
 *
 * Workflow:
 * 1. The function runs in a loop and listens for messages from the `messageChan` channel.
 * 2. Each message is validated using the `processMessage` function:
 *    - If valid, the message is published to the `outputTopic`.
 *    - If invalid, the message is enriched with metadata and published to the `dlqTopic`.
 * 3. Retry logic is applied when publishing messages, ensuring up to three attempts with a delay between retries.
 * 4. The function terminates gracefully when the `ctx.Done` signal is received or the channel is closed.
 *
 * @param ctx: A `context.Context` instance used for managing the lifecycle of the worker goroutine. The function 
 *             exits when the context is canceled, ensuring clean shutdown.
 * @param messageChan: A read-only channel (`<-chan`) through which Kafka messages are received from the main consumer loop.
 * @param producer: An interface (`ProducerInterface`) that abstracts Kafka producer functionality. Used to send 
 *                  processed messages to the appropriate topic.
 * @param outputTopic: A string representing the name of the Kafka topic where valid messages are published.
 * @param dlqTopic: A string representing the name of the Kafka Dead Letter Queue (DLQ) topic where invalid messages 
 *                  are published for troubleshooting or reprocessing.
 * 
 * @return: void
 * - This function does not return any value.*
 *
 * Notes:
 * - This function assumes that the `ProducerInterface` implementation handles low-level Kafka producer behavior.
 * - Metrics are updated for each processed message to track success and DLQ counts using Prometheus.
 */
func processMessages(ctx context.Context, messageChan <-chan *kafka.Message, producer ProducerInterface, outputTopic, dlqTopic string) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messageChan:
			if !ok {
				return
			}

			processedMsg, valid := processMessage(msg.Value)
			if !valid {
				dlqMessage := map[string]interface{}{
					"error": map[string]interface{}{
						"reason":   "Invalid message structure",
						"message":  msg.Value,
						"metadata": map[string]interface{}{
							"topic":     *msg.TopicPartition.Topic,
							"partition": msg.TopicPartition.Partition,
							"offset":    msg.TopicPartition.Offset,
						},
					},
				}

				delay := time.Second // Defined delay for retries
				dlqBytes, err := json.Marshal(dlqMessage)

				if err != nil {
					log.Printf("Failed to marshal DLQ message: %v", err)
					continue
				}

				log.Printf("Invalid message sent to DLQ: Topic: %s, Partition: %d, Offset: %d",
					*msg.TopicPartition.Topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset)

				err = publishWithRetry(producer, dlqTopic, dlqBytes, 3, delay)
				if err != nil {
					log.Printf("Failed to publish to DLQ after retries: %v", err)
				}

				kafkaMessagesProcessed.WithLabelValues("dlq").Inc()
				continue
			}

			err := publishWithRetry(producer, outputTopic, processedMsg, 3, time.Second)
			if err != nil {
				log.Printf("Failed to publish message to output topic: %v", err)
				continue
			}

			kafkaMessagesProcessed.WithLabelValues("success").Inc()

			log.Printf("Processed and published message: %s", string(processedMsg))
		}
	}
}

/**
 * processMessage:
 *
 * Validates and processes a single Kafka message.
 *
 * Workflow:
 * 1. The input raw message payload (JSON) is unmarshalled into a `Message` struct.
 * 2. The validity of the message is checked using the `isValidMessage` function:
 *    - If the message is invalid (e.g., missing required fields), the function logs the error and returns a 
 *      `nil` byte array and `false` for the validity flag.
 * 3. If valid, the `Locale` field of the message is converted to lowercase.
 * 4. The processed message is enriched with a timestamp and marshalled into a JSON format.
 * 5. The function returns the marshalled processed message and a `true` validity flag.
 *
 * @param value: A byte array containing the raw message payload in JSON format.
 *               - Example payload: {"id": "123", "content": "example", "locale": "EN"}
 *
 * @return: A tuple containing:
 *          1. A byte array representing the processed message in JSON format, enriched with a processing timestamp.
 *             - Example processed payload: {"Message": {"id": "123", "content": "example", "locale": "en"}, "ProcessedAt": "2024-12-15T15:04:05Z"}
 *          2. A boolean flag indicating whether the message was valid and successfully processed:
 *             - `true` if the message was valid and processed successfully.
 *             - `false` if the message was invalid or could not be processed.
 *
 * Notes:
 * - Invalid messages are not processed further and should typically be sent to a Dead Letter Queue (DLQ).
 * - JSON marshalling and unmarshalling ensure the input and output messages adhere to structured formats.
 * - The function is designed to be used as part of a Kafka consumer pipeline.
 */
func processMessage(value []byte) ([]byte, bool) {
	var msg Message
	if err := json.Unmarshal(value, &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return nil, false
	}

	if !isValidMessage(msg) {
		return nil, false
	}

	msg.Locale = strings.ToLower(msg.Locale)

	processed := ProcessedMessage{
		Message:     msg,
		ProcessedAt: time.Now().Format(time.RFC3339),
	}

	processedBytes, err := json.Marshal(processed)
	if err != nil {
		log.Printf("Failed to marshal processed message: %v", err)
		return nil, false
	}

	return processedBytes, true
}

/**
 * isValidMessage:
 *
 * Validates a message to ensure it meets the criteria for processing.
 *
 * Workflow:
 * 1. Checks if required fields (`UserID`, `AppVersion`, `DeviceType`) are present and non-empty.
 *    - Logs an error and returns `false` if any required field is missing.
 * 2. Ensures the `AppVersion` meets the minimum required version ("2.0.0").
 *    - Logs an error and returns `false` if the version is below the threshold.
 * 3. Verifies that the IP address is not a private IP address.
 *    - Logs an error and returns `false` if the IP is private.
 * 4. If all validations pass, the function returns `true`, indicating the message is valid.
 *
 * @param msg: The `Message` object to be validated. This struct contains the following fields:
 *             - `UserID` (string): The unique identifier for the user. Must be non-empty.
 *             - `AppVersion` (string): The version of the application. Must meet the minimum required version ("2.0.0").
 *             - `DeviceType` (string): The type of device used (e.g., "mobile", "desktop"). Must be non-empty.
 *             - `IP` (string): The IP address of the user. Must not be a private IP.
 *
 * @return: A boolean value indicating the validity of the message:
 *          - `true`: If the message passes all validation checks.
 *          - `false`: If the message fails any validation check.
 *
 * Notes:
 * - This function is critical for filtering out messages that do not meet the processing requirements.
 * - Messages failing validation are logged for debugging and monitoring purposes.
 */
func isValidMessage(msg Message) bool {
	if msg.UserID == "" {
		log.Println("Skipping message due to missing UserID")
		return false
	}
	if msg.AppVersion == "" {
		log.Println("Skipping message due to missing AppVersion")
		return false
	}
	if msg.DeviceType == "" {
		log.Println("Skipping message due to missing DeviceType")
		return false
	}
	if msg.AppVersion < "2.0.0" {
		log.Println("Skipping message with app_version < 2.0.0")
		return false
	}
	if isPrivateIP(msg.IP) {
		log.Println("Skipping message from private IP address")
		return false
	}
	return true
}

/**
 * publishWithRetry:
 *
 * Publishes a message to a Kafka topic with retry attempts in case of failure.
 *
 * Workflow:
 * 1. Tries to send a message to a Kafka topic using the provided Kafka producer.
 * 2. If the message fails to be sent, it retries up to the specified number of times (`retries`).
 * 3. After each failed attempt, it waits for a specified delay (`delay`) before trying again.
 * 4. If all retry attempts fail, it returns an error indicating the failure.
 * 5. If the message is successfully sent, it returns `nil`, indicating no error.
 *
 * @param producer: `ProducerInterface` - An interface to the Kafka producer object used for sending messages.
 *                The producer handles the actual communication with the Kafka broker.
 * @param topic: `string` - The name of the Kafka topic to which the message will be sent.
 *                Kafka topics act as message channels where messages are produced and consumed.
 * @param message: `[]byte` - The payload of the message to be sent, usually in JSON format or another structured format.
 *                This is the data that will be transmitted to the Kafka topic.
 * @param retries: `int` - The maximum number of retry attempts if the message fails to be published.
 *                This value determines how many times the function will attempt to resend the message.
 * @param delay: `time.Duration` - The duration to wait between retry attempts. A typical value might be 1 second (`time.Second`).
 *                This ensures that the retries are spaced out, giving time for temporary issues to resolve.
 *
 * @return: `error` - If the message is successfully sent, the function returns `nil`.
 *                    If all retry attempts fail, it returns an error message describing the failure reason.
 *                    The error message includes the number of attempts made and the last error encountered.
 * 
 * Notes:
 * - This function is useful when there are temporary issues with the Kafka broker or network that might prevent immediate delivery.
 * - The retry mechanism ensures that transient failures do not result in immediate message loss.
 */
func publishWithRetry(producer ProducerInterface, topic string, message []byte, retries int, delay time.Duration) error {
	var err error
	for attempt := 0; attempt < retries; attempt++ {
		kafkaMessage := &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: kafka.PartitionAny,
				Offset:    kafka.OffsetEnd,
			},
			Value: message,
		}

		err = producer.Produce(kafkaMessage, nil)
		if err == nil {
			return nil
		}

		fmt.Printf("Error producing message (attempt %d/%d): %v\n", attempt+1, retries, err)
		time.Sleep(delay)
	}

	return fmt.Errorf("failed to produce message after %d attempts: %v", retries, err)
}

/**
 * isPrivateIP:
 *
 * Checks if a given IP address is a private IP address.
 * 
 * Private IP addresses are used within local networks and are not routable on the public internet.
 * Common ranges for private IP addresses include:
 * - 10.0.0.0 to 10.255.255.255
 * - 172.16.0.0 to 172.31.255.255
 * - 192.168.0.0 to 192.168.255.255
 *
 * This function checks if the provided IP address falls within these ranges.
 * Private IPs are often used for internal network communication, and it's useful to identify them to avoid external exposure.
 *
 * @param ip: `string` - The IP address to check. This is the address in string format (e.g., "192.168.1.1").
 *
 * @return: `bool` - Returns `true` if the IP address is private, and `false` if it is not.
 *                    A private IP address is one that falls within the specified private IP ranges.
 *                    If the IP is not private (i.e., it's a public IP), the function will return `false`.
 *
 * Notes:
 * - The `net.ParseIP` function is used to convert the string into an `IP` type, which is then checked to see if it's in the private IP range.
 * - The method `IsPrivate` is a built-in function that returns true for IPs within the private address ranges.
 * - This function does not handle invalid IP formats (e.g., malformed IP addresses); it assumes the input is a valid IP string.
 */
func isPrivateIP(ip string) bool {
    parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.IsPrivate()
}

/**
 * handleSignals:
 *
 * Handles termination signals (SIGINT, SIGTERM) and performs a graceful shutdown of the Kafka consumer and producer.
 *
 * When the program receives termination signals such as SIGINT (interrupt) or SIGTERM (terminate), this function
 * ensures that the consumer and producer are properly closed to avoid data loss or resource leakage. It also
 * cancels ongoing operations and ensures that all connections are gracefully terminated.
 *
 * The function listens for signals using a channel, and when a signal is received, it initiates the shutdown process.
 *
 * @param cancel: `context.CancelFunc` - A function that cancels ongoing operations, typically used to stop active
 *                 background tasks, such as processing or consuming messages.
 * @param consumer: `ConsumerInterface` - An interface to the Kafka consumer object. It is used for consuming messages
 *                  from Kafka topics. This interface is decoupled from a specific Kafka implementation to make the code
 *                  easier to unit test and mock.
 * @param producer: `ProducerInterface` - An interface to the Kafka producer object. It is used for sending messages
 *                  to Kafka topics. Like the consumer, this interface is decoupled for easier unit testing and mocking.
 *
 * @return: `void` - This function does not return any values. It performs the shutdown and cleanup operations
 *                  without returning anything to the caller.
 *
 * Notes:
 * - The function listens for specific system signals (`SIGINT` and `SIGTERM`), which are used to indicate that the
 *   program should shut down. These signals are commonly sent when the user interrupts the program or when the
 *   system is shutting down.
 * - The `cancel` function is used to signal that ongoing operations should stop. For example, this could be used
 *   to stop background tasks that are processing messages or consuming from Kafka.
 * - The `Close` method on both the producer and consumer is called to close the connections to Kafka and ensure
 *   that no further messages are sent or consumed.
 */
func handleSignals(cancel context.CancelFunc, consumer ConsumerInterface, producer ProducerInterface) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	cancel()

	// Close producer and consumer
	if err := producer.Close(); err != nil {
		log.Printf("Failed to close producer: %v", err)
	}
	if err := consumer.Close(); err != nil {
		log.Printf("Failed to close consumer: %v", err)
	}

	log.Println("Kafka consumer and producer closed successfully.")
}

/**
 * startMetricsServer:
 *
 * Starts an HTTP server that exposes application metrics for Prometheus monitoring.
 *
 * This function sets up an HTTP endpoint at `/metrics`, which Prometheus scrapes to collect
 * metrics about the application. The metrics include data related to the application's
 * performance, health, and usage, which can then be monitored and visualized using Prometheus
 * and tools like Grafana. The server listens on port 9090.
 *
 * @param: None
 *
 * @return: void
 * - This function does not return any value. It starts the metrics server in the background.
 *
 * Notes:
 * - The function sets up an HTTP route that exposes the `/metrics` endpoint, which is
 *   a standard Prometheus endpoint for exposing metrics data.
 * - The `http.ListenAndServe` function runs the server on port 9090, and it is executed
 *   in a separate goroutine to allow the main program to continue running without blocking.
 * - If an error occurs while starting the server, `log.Fatal` is called, which logs the
 *   error and terminates the program.
 */
func startMetricsServer() {
    http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":9090", nil))
	}()
}

