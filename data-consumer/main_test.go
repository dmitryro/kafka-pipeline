package main

import (
    "errors"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// MockProducer is a mock implementation of the kafka.Producer interface
type MockProducer struct {
    mock.Mock
}

func (m *MockProducer) Produce(message *kafka.Message, deliveryChan chan kafka.Event) error {
    args := m.Called(message, deliveryChan)
    return args.Error(0)
}

func (m *MockProducer) Close() error {
    args := m.Called()
    return args.Error(0)
}

func TestIsValidMessage(t *testing.T) {
    // Test case 1: Valid message with all required fields
    validMessage := []byte(`{"user_id":"user1", "app_version":"2.1.0", "device_type":"android", "ip":"192.168.1.1", "locale":"en", "device_id":"device1", "timestamp":1694479551}`)
    valid := isValidMessage(validMessage)
    assert.True(t, valid, "Expected message to be valid")

    // Test case 2: Invalid message (missing 'user_id' field)
    invalidMessageMissingUserID := []byte(`{"app_version":"2.1.0", "device_type":"android", "ip":"192.168.1.1", "locale":"en", "device_id":"device1", "timestamp":1694479551}`)
    valid = isValidMessage(invalidMessageMissingUserID)
    assert.False(t, valid, "Expected message to be invalid due to missing 'user_id'")

    // Test case 3: Invalid message (missing 'timestamp' field)
    invalidMessageMissingTimestamp := []byte(`{"user_id":"user1", "app_version":"2.1.0", "device_type":"android", "ip":"192.168.1.1", "locale":"en", "device_id":"device1"}`)
    valid = isValidMessage(invalidMessageMissingTimestamp)
    assert.False(t, valid, "Expected message to be invalid due to missing 'timestamp'")

    // Test case 4: Invalid message (invalid JSON format)
    invalidJSONMessage := []byte(`{"user_id":"user1", "app_version":"2.1.0", "device_type":"android", "ip":"192.168.1.1", "locale":"en", "device_id":"device1", "timestamp"}`)
    valid = isValidMessage(invalidJSONMessage)
    assert.False(t, valid, "Expected message to be invalid due to incorrect JSON format")

    // Test case 5: Invalid message (incorrect field types)
    invalidMessageIncorrectFieldType := []byte(`{"user_id":"user1", "app_version":"2.1.0", "device_type":"android", "ip":"192.168.1.1", "locale":"en", "device_id":"device1", "timestamp":"invalid"}`)
    valid = isValidMessage(invalidMessageIncorrectFieldType)
    assert.False(t, valid, "Expected message to be invalid due to 'timestamp' field having an incorrect type")
}

func TestProcessMessage(t *testing.T) {
    // Test case 1: Valid message
    validMessage := []byte(`{"user_id":"user1", "app_version":"2.1.0", "device_type":"android", "ip":"192.168.1.1", "locale":"en", "device_id":"device1", "timestamp":1694479551}`)
    processedMessage, valid := processMessage(validMessage)
    assert.True(t, valid)
    assert.NotNil(t, processedMessage)
    assert.Equal(t, "user1", processedMessage.UserID)  // Example field validation
    assert.Equal(t, "android", processedMessage.DeviceType)

    // Test case 2: Invalid message (missing required field)
    invalidMessage := []byte(`{"app_version":"2.1.0", "device_type":"android", "ip":"192.168.1.1", "locale":"en", "device_id":"device1", "timestamp":1694479551}`)
    processedMessage, valid = processMessage(invalidMessage)
    assert.False(t, valid)
}

func TestPublishWithRetry(t *testing.T) {
    // Create a mock producer
    mockProducer := new(MockProducer)
    mockProducer.On("Produce", mock.Anything, mock.Anything).Return(errors.New("test error")).Once()  // Simulate failure
    mockProducer.On("Produce", mock.Anything, mock.Anything).Return(nil).Once()  // Simulate success

    // Test the retry logic
    err := publishWithRetry(mockProducer, "test-topic", []byte("test message"), 2)

    // Verify the mock producer was called twice
    mockProducer.AssertExpectations(t)
    assert.Nil(t, err)  // Ensure there is no error after retrying
}

func TestIsPrivateIP(t *testing.T) {
    // Test private IP
    assert.True(t, isPrivateIP("192.168.1.1"))

    // Test public IP
    assert.False(t, isPrivateIP("8.8.8.8"))

    // Test loopback IP
    assert.True(t, isPrivateIP("127.0.0.1"))

    // Test a reserved IP range (not necessarily private)
    assert.True(t, isPrivateIP("169.254.1.1"))  // APIPA address
}

func TestClose(t *testing.T) {
    // Create a mock producer
    mockProducer := new(MockProducer)
    mockProducer.On("Close").Return(nil)

    // Test closing the producer
    err := mockProducer.Close()
    assert.NoError(t, err)

    // Verify Close was called
    mockProducer.AssertExpectations(t)
}

func TestHandleError(t *testing.T) {
    // Assuming handleError does some logging or custom error handling
    err := errors.New("sample error")
    // You would typically test whether logging is called here if the function does something like that
    // For now, just make sure the function doesn't panic

    assert.NotPanics(t, func() {
        handleError(err)
    })

    // If handleError has any side effects, assert those as well (e.g., logging)
}
