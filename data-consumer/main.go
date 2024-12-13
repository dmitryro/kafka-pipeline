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
	UserID     string `json:"user_id"`
	AppVersion string `json:"app_version"`
	DeviceType string `json:"device_type"`
	IP         string `json:"ip"`
	Locale     string `json:"locale"`
	DeviceID   string `json:"device_id"`
	Timestamp  int64  `json:"timestamp"`
}

type ProcessedMessage struct {
	Message
	ProcessedAt string `json:"processed_at"`
}

func main() {
	inputTopic := os.Getenv("INPUT_TOPIC")
	outputTopic :=  os.Getenv("OUTPUT_TOPIC")
	dlqTopic :=  os.Getenv("DLQ_TOPIC")

    if inputTopic == "" || outputTopic == "" || dlqTopic == "" {
        log.Fatal("One or more required environment variables are not set.")
    }

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

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("BOOTSTRAP_SERVERS"),
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}
	defer producer.Close()

	err = consumer.Subscribe(inputTopic, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topic: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	messageChan := make(chan *kafka.Message, 100)

	// Signal handling for graceful shutdown
	go handleSignals(cancel, consumer, producer)

	// Goroutine to process messages with a worker pool
	workerPoolSize := 10
	wg.Add(workerPoolSize)
	for i := 0; i < workerPoolSize; i++ {
		go func() {
			defer wg.Done()
			processMessages(ctx, messageChan, producer, outputTopic, dlqTopic)
		}()
	}

	log.Println("Consumer is running...")

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
	            // Commit offsets after processing
	        case kafka.Error:
	            log.Printf("Consumer error: %v", ev)
	        }
	    }
	}
}

func processMessages(ctx context.Context, messageChan <-chan *kafka.Message, producer *kafka.Producer, outputTopic, dlqTopic string) {
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
	var msg Message
	if err := json.Unmarshal(value, &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return nil, false
	}

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

func publishWithRetry(producer *kafka.Producer, topic string, message []byte, maxRetries int) {
	for i := 0; i < maxRetries; i++ {
		err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          message,
		}, nil)
		if err == nil {
			log.Printf("Message published to topic %s: %s", topic, string(message))
			return
		}
		log.Printf("Retrying to produce message to topic %s (attempt %d/%d): %v", topic, i+1, maxRetries, err)
		time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second) // Exponential backoff
	}

	log.Printf("Failed to produce message to topic %s after %d attempts", topic, maxRetries)
}

func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	return parsedIP.IsPrivate()
}

func handleSignals(cancel context.CancelFunc, consumer *kafka.Consumer, producer *kafka.Producer) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	cancel()
	consumer.Close()
	producer.Close()
}

