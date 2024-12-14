package main

import (
        "context"
        "encoding/json"
        "log"
        "math"
        "net"
        "os"
        "os/signal"
        "strings"
        "sync"
        "syscall"
        "time"

        "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Message struct {
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
        UserID     string `json:"user_id"`
        AppVersion string `json:"app_version"`
        DeviceType string `json:"device_type"`
        IP         string `json:"ip"`
        Locale     string `json:"locale"`
        DeviceID   string `json:"device_id"`
        Timestamp  int64  `json:"timestamp"`
}


type ProcessedMessage struct {
        /**
         * ProcessedMessage:
         *
         * Represents a processed message, including the original message and a timestamp of processing.
         *
         * Fields:
         *   - Message: The original message received from the Kafka topic.
         *   - ProcessedAt: Timestamp indicating when the message was processed.
         */
        Message
        ProcessedAt string `json:"processed_at"`
}


func main() {
        /**
         * Main function:
         *
         * Initializes the Kafka consumer and producer, subscribes to the input topic, sets up signal handling, launches worker goroutines, and starts the main consumer loop.
         */
        // Retrieve environment variables for Kafka configuration and topic names
        inputTopic := os.Getenv("INPUT_TOPIC")
        outputTopic := os.Getenv("OUTPUT_TOPIC")
        dlqTopic := os.Getenv("DLQ_TOPIC")

        if inputTopic == "" || outputTopic == "" || dlqTopic == "" {
                log.Fatal("One or more required environment variables are not set.")
        }

        // Create a new Kafka consumer with the specified configuration
        consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
                "bootstrap.servers": os.Getenv("BOOTSTRAP_SERVERS"),
                "group.id":          os.Getenv("CONSUMER_GROUP"),
                "auto.offset.reset": os.Getenv("AUTO_OFFSET_RESET"),
                "enable.auto.commit": os.Getenv("ENABLE_AUTO_COMMT"), // Explicit offset management
        })
        if err != nil {
                log.Fatalf("Failed to create consumer: %s", err)
        }
        defer consumer.Close()

        // Create a new Kafka producer with the specified configuration
        producer, err := kafka.NewProducer(&kafka.ConfigMap{
                "bootstrap.servers": os.Getenv("BOOTSTRAP_SERVERS"),
        })
        if err != nil {
                log.Fatalf("Failed to create producer: %s", err)
        }
        defer producer.Close()

        // Subscribe the consumer to the input topic
        err = consumer.Subscribe(inputTopic, nil)
        if err != nil {
                log.Fatalf("Failed to subscribe to topic: %s", err)
        }

        // Create a context for graceful shutdown and a wait group to synchronize goroutines
        ctx, cancel := context.WithCancel(context.Background())
        wg := &sync.WaitGroup{}
        messageChan := make(chan *kafka.Message, 100)

        // Set up a signal handler for graceful shutdown
        go handleSignals(cancel, consumer, producer)

        // Create a worker pool to process messages concurrently
        workerPoolSize := 10
        wg.Add(workerPoolSize)
        for i := 0; i < workerPoolSize; i++ {
                go func() {
                        defer wg.Done()
                        processMessages(ctx, messageChan, producer, outputTopic, dlqTopic)
                }()
        }

        log.Println("Consumer is running...")

        // Main consumer loop to poll messages and pass them to the worker pool
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
                                messageChan <- ev
                        case kafka.Error:
                                log.Printf("Consumer error: %v", ev)
                        }
                }
        }
}


func processMessages(ctx context.Context, messageChan <-chan *kafka.Message, producer *kafka.Producer, outputTopic, dlqTopic string) {
        /**
         * ProcessMessages:
         *
         * Processes messages from the message channel and sends them to the appropriate topic.
         *
         * @param ctx: Context for managing worker goroutine lifecycle.
         * @param messageChan: Channel for receiving messages from the main consumer loop.
         * @param producer: Kafka producer instance for sending processed messages.
         * @param outputTopic: Name of the output topic for valid messages.
         * @param dlqTopic: Name of the Dead Letter Queue (DLQ) topic for invalid messages.
         */
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
                                log.Printf("Invalid message sent to DLQ: Topic: %s, Partition: %d, Offset: %d",
                                        *msg.TopicPartition.Topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset)
                                publishWithRetry(producer, dlqTopic, msg.Value, 3)
                                continue
                        }

                        publishWithRetry(producer, outputTopic, processedMsg, 3)
                }
        }
}


func processMessage(value []byte) ([]byte, bool) {
        /**
         * ProcessMessage:
         *
         * Validates and processes a single message.
         *
         * @param value: Raw byte array containing the message payload (JSON).
         *
         * @return: A tuple containing the processed message (if valid) and a boolean indicating validity.
         */
        var msg Message
        if err := json.Unmarshal(value, &msg); err != nil {
                log.Printf("Failed to unmarshal message: %v", err)
                return nil, false
        }

        // Validate message fields
        if msg.UserID == "" {
                log.Println("Skipping message due to missing UserID")
                return nil, false
        }

        if msg.AppVersion == "" {
                log.Println("Skipping message due to missing AppVersion")
                return nil, false
        }

        if msg.DeviceType == "" {
                log.Println("Skipping message due to missing DeviceType")
                return nil, false
        }

        if msg.AppVersion < "2.0.0" {
                log.Println("Skipping message with app_version < 2.0.0")
                return nil, false
        }

        if isPrivateIP(msg.IP) {
                log.Println("Skipping message from private IP address")
                return nil, false
        }

        // Normalize locale
        msg.Locale = strings.ToLower(msg.Locale)

        // Create a processed message with additional metadata
        processed := ProcessedMessage{
                Message:     msg,
                ProcessedAt: time.Now().Format(time.RFC3339),
        }

        // Marshal the processed message to bytes
        processedBytes, err := json.Marshal(processed)
        if err != nil {
                log.Printf("Failed to marshal processed message: %v", err)
                return nil, false
        }

        return processedBytes, true
}


func publishWithRetry(producer *kafka.Producer, topic string, message []byte, maxRetries int) {
        /**
         * PublishWithRetry:
         *
         * Publishes a message to a Kafka topic with retries.
         *
         * @param producer: Kafka producer instance for sending messages.
         * @param topic: Name of the target Kafka topic.
         * @param message: Message payload to be sent.
         * @param maxRetries: Maximum number of retries for failed delivery attempts.
         */
        for i := 0; i < maxRetries; i++ {
                err := producer.Produce(&kafka.Message{
                        TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
                        Value:           message,
                }, nil)
                if err == nil {
                        log.Printf("Message published to topic %s: %s", topic, string(message))
                        return
                }

                // Check for specific error messages or codes to determine retry logic
                if strings.Contains(err.Error(), "network timeout") {
                        log.Printf("Retrying to produce message to topic %s (attempt %d/%d): %v", topic, i+1, maxRetries, err)
                } else {
                        // Log critical errors and consider exiting the program
                        log.Fatalf("Failed to produce message after %d attempts: %v", maxRetries, err)
                }

                time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second) // Exponential backoff
        }

        log.Printf("Failed to produce message to topic %s after %d attempts", topic, maxRetries)
}


func isPrivateIP(ip string) bool {
        /**
         * IsPrivateIP:
         *
         * Checks if a given IP address is a private IP address.
         *
         * @param ip: The IP address to check.
         *
         * @return: True if the IP is private, false otherwise.
         */
        parsedIP := net.ParseIP(ip)
        if parsedIP == nil {
                return false
        }
        return parsedIP.IsPrivate()
}


func handleSignals(cancel context.CancelFunc, consumer *kafka.Consumer, producer *kafka.Producer) {
        /**
         * HandleSignals:
         *
         * Handles termination signals (SIGINT, SIGTERM) and performs a graceful shutdown.
         *
         * @param cancel: Function to cancel the ongoing operations.
         * @param consumer: Kafka consumer instance.
         * @param producer: Kafka producer instance.
         */
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
        <-sigChan

        log.Println("Shutting down gracefully...")
        cancel()

        // Gracefully close the consumer and producer without expecting a return value
        consumer.Close()
        producer.Close()
}
