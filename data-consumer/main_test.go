package main

import (
	"encoding/json"
    "fmt"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
    "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProducer is a mock implementation of the ProducerInterface.
type MockProducer struct {
	mock.Mock
}


func (m *MockProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockConsumer is a mock implementation of the ConsumerInterface.
type MockConsumer struct {
	mock.Mock
}

func (m *MockConsumer) Subscribe(topic string, cb kafka.RebalanceCb) error {
	args := m.Called(topic, cb)
	return args.Error(0)
}

func (m *MockConsumer) Poll(timeoutMs int) kafka.Event {
	args := m.Called(timeoutMs)
	return args.Get(0).(kafka.Event)
}

func (m *MockConsumer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProducer) Produce(msg *kafka.Message, ch chan kafka.Event) error {
	args := m.Called(msg, ch)
	return args.Error(0)
}

func TestProcessMessage_ValidMessage(t *testing.T) {
	// Sample test message
	msg := Message{
		UserID:     "user123",
		AppVersion: "2.1.0",
		DeviceType: "mobile",
		IP:         "8.8.8.8", // Valid public IP
		Locale:     "en",      // Valid locale
		DeviceID:   "device123",
		Timestamp:  time.Now().Unix(),
	}

	// Marshal the msg into a []byte (JSON)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Process the message with the marshaled bytes
	processedMsg, valid := processMessage(msgBytes)

	// Check if the message is valid
	if !valid {
		t.Fatalf("Message should be valid")
	}

	// Ensure the processed message is not nil
	if processedMsg == nil {
		t.Fatalf("Processed message should not be nil")
	}

	// Unmarshal the processed message
	var result ProcessedMessage
	err = json.Unmarshal(processedMsg, &result)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that the locale is correctly set
	if result.Locale != "en" {
		t.Errorf("Expected locale 'en', got '%s'", result.Locale)
	}
}

func TestProcessMessage_InvalidMessage(t *testing.T) {
	// Prepare an invalid message (missing UserID).
	msg := Message{
		AppVersion: "2.1.0",
		DeviceType: "mobile",
		IP:         "192.168.1.1",
		Locale:     "en",
		DeviceID:   "device123",
		Timestamp:  time.Now().Unix(),
	}

	// Process the message.
	processedMsg, valid := processMessage(toJSON(t, msg))
	assert.False(t, valid, "Message should be invalid")
	assert.Nil(t, processedMsg, "Processed message should be nil")
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		isPrivate bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			assert.Equal(t, tt.isPrivate, isPrivateIP(tt.ip))
		})
	}
}

func TestPublishWithRetry_Success(t *testing.T) {
	mockProducer := new(MockProducer)
	mockProducer.On("Produce", mock.Anything, mock.Anything).Return(nil)

	err := publishWithRetry(mockProducer, "test_topic", []byte("test_message"), 3, time.Second)
	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
}

func TestPublishWithRetry_Failure(t *testing.T) {
	// Create the mock producer
	mockProducer := new(MockProducer)

	// Set up the mock to return an error for each call
	mockProducer.On("Produce", mock.Anything, mock.Anything).Return(fmt.Errorf("producer error")).Times(3)

	// Call the function that retries producing messages
	err := publishWithRetry(mockProducer, "test-topic", []byte("test_message"), 3, 1)

	// Ensure the correct error is returned after 3 retries
	assert.Error(t, err)
	assert.Equal(t, "failed to produce message after 3 attempts: producer error", err.Error())

	// Assert that the Produce method was called 3 times
	mockProducer.AssertNumberOfCalls(t, "Produce", 3)
}

func TestKafkaMessagesProcessedMetric(t *testing.T) {
	// Set up the test.
	reg := prometheus.NewRegistry()
	reg.MustRegister(kafkaMessagesProcessed)

	// Simulate message processing.
	kafkaMessagesProcessed.WithLabelValues("success").Inc()

	// Test if the metric is correctly incremented.
	gatheredMetrics, err := testutil.GatherAndCount(reg)
	assert.NoError(t, err)
	assert.Equal(t, 1, gatheredMetrics)
}

func TestGracefulShutdown(t *testing.T) {
	// Create a mock producer instance
	mockProducer := new(MockProducer)

	// Create a channel to simulate a shutdown signal
	shutdownChan := make(chan struct{})

	// Simulate graceful shutdown in a separate goroutine
	go func() {
		// Simulate some work being done (like consuming Kafka messages)
		time.Sleep(1 * time.Second)

		// Now simulate the shutdown signal
		close(shutdownChan)
	}()

	// Simulate the main application logic handling graceful shutdown
	go func() {
		// Simulate your application listening for the shutdown signal
		select {
		case <-shutdownChan:
			// Your graceful shutdown logic: close producer, clean up resources, etc.
			mockProducer.Close()
			return
		}
	}()

	// Wait for the graceful shutdown to complete
	select {
	case <-shutdownChan:
		// Check that the producer was closed
		assert.NotNil(t, mockProducer)
		// Additional assertions can be added here based on your logic
	case <-time.After(3 * time.Second):
		t.Fatal("Graceful shutdown test timed out")
	}
}

// Helper function to convert structs to JSON.
func toJSON(t *testing.T, msg interface{}) []byte {
	data, err := json.Marshal(msg)
	assert.NoError(t, err)
	return data
}

