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

// Define the ProducerInterface for flexibility
type ProducerInterface interface {
	Produce(*kafka.Message, chan kafka.Event) error
}

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

// Prometheus Metrics
var (
	kafkaMessagesProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_processed_total",
			Help: "Total number of Kafka messages processed.",
		},
		[]string{"result"},
	)
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(kafkaMessagesProcessed)
}

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
		"session.timeout.ms":  os.Getenv("KAFKA_SESSION_TIMEOUT"),
		"socket.timeout.ms":  os.Getenv("KAFKA_SOCKET_TIMEOUT"),
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

func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.IsPrivate()
}

func handleSignals(cancel context.CancelFunc, consumer *kafka.Consumer, producer *kafka.Producer) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	cancel()

	consumer.Close()
	producer.Close()

	log.Println("Kafka consumer and producer closed successfully.")
}

func startMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":9090", nil))
	}()
}

