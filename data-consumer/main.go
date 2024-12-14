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
	/** Represents a message received from the Kafka topic. */
	UserID     string `json:"user_id"`
	AppVersion string `json:"app_version"`
	DeviceType string `json:"device_type"`
	IP         string `json:"ip"`
	Locale     string `json:"locale"`
	DeviceID   string `json:"device_id"`
	Timestamp  int64  `json:"timestamp"`
}

type ProcessedMessage struct {
	/** Represents a processed message, including the original message and timestamp of processing. */
	Message
	ProcessedAt string `json:"processed_at"`
}

func main() {
	/** Main function initializing Kafka consumer and producer, subscribing to the input topic, and handling signals. */
	inputTopic := os.Getenv("INPUT_TOPIC")
	outputTopic := os.Getenv("OUTPUT_TOPIC")
	dlqTopic := os.Getenv("DLQ_TOPIC")

	if inputTopic == "" || outputTopic == "" || dlqTopic == "" {
		log.Fatal("One or more required environment variables are not set.")
	}

	// Initialize the consumer
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("BOOTSTRAP_SERVERS"),
		"group.id":          os.Getenv("CONSUMER_GROUP"),
		"auto.offset.reset": os.Getenv("AUTO_OFFSET_RESET"),
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %s", err)
	}
	defer consumer.Close()

	// Initialize the producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("BOOTSTRAP_SERVERS"),
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}
	defer producer.Close()

	// Subscribe to the input topic
	err = consumer.Subscribe(inputTopic, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topic: %s", err)
	}

	// Set up for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	messageChan := make(chan *kafka.Message, 100)

	go handleSignals(cancel, consumer, producer)

	// Worker pool to process messages concurrently
	workerPoolSize := 10
	wg.Add(workerPoolSize)
	for i := 0; i < workerPoolSize; i++ {
		go func() {
			defer wg.Done()
			processMessages(ctx, messageChan, producer, outputTopic, dlqTopic)
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

func processMessages(ctx context.Context, messageChan <-chan *kafka.Message, producer *kafka.Producer, outputTopic, dlqTopic string) {
	/** Processes messages from the channel and sends them to the appropriate topic. */
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

				dlqBytes, err := json.Marshal(dlqMessage)
				if err != nil {
					log.Printf("Failed to marshal DLQ message: %v", err)
					continue
				}

				log.Printf("Invalid message sent to DLQ: Topic: %s, Partition: %d, Offset: %d", 
					*msg.TopicPartition.Topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset)

				// Retry publishing to DLQ
				publishWithRetry(producer, dlqTopic, dlqBytes, 3)
				continue
			}

			// Send valid processed message to output topic
			publishWithRetry(producer, outputTopic, processedMsg, 3)
		}
	}
}

func processMessage(value []byte) ([]byte, bool) {
	/** Validates and processes a single message. */
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

func isValidMessage(msg Message) bool {
	/** Validates message structure. */
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

func publishWithRetry(producer *kafka.Producer, topic string, message []byte, maxRetries int) {
	/** Publishes a message with retry logic in case of failure. */
	for i := 0; i < maxRetries; i++ {
		err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          message,
		}, nil)
		if err == nil {
			log.Printf("Message published to topic %s: %s", topic, string(message))
			return
		}

		if strings.Contains(err.Error(), "network timeout") {
			log.Printf("Retrying to produce message to topic %s (attempt %d/%d): %v", topic, i+1, maxRetries, err)
		} else {
			log.Fatalf("Failed to produce message after %d attempts: %v", maxRetries, err)
		}

		time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
	}

	log.Printf("Failed to produce message to topic %s after %d attempts", topic, maxRetries)
}

func isPrivateIP(ip string) bool {
	/** Checks if an IP is private. */
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.IsPrivate()
}

func handleSignals(cancel context.CancelFunc, consumer *kafka.Consumer, producer *kafka.Producer) {
	/** Handles termination signals and performs graceful shutdown. */
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	cancel()

	consumer.Close()
	producer.Close()

	log.Println("Kafka consumer and producer closed successfully.")
}

