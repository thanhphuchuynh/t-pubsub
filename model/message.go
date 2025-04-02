package model

import (
	"time"

	"github.com/google/uuid"
)

// Message represents a message in the pub/sub system
type Message struct {
	ID        string
	Content   interface{}
	Timestamp time.Time
}

// Pure function to create a new message with a generated ID
func NewMessage(content interface{}) Message {
	return Message{
		ID:        GenerateUniqueID(),
		Content:   content,
		Timestamp: time.Now(),
	}
}

// ValidType checks if the message content is of type T and returns it
// This is a generic function rather than a method since Go methods cannot have type parameters
func ValidType[T any](m *Message, t T) (bool, T) {
	if msg, ok := m.Content.(T); ok {
		return true, msg
	}
	return false, *new(T)
}

// Pure function to create a new message with a custom ID
func NewMessageWithID(id string, content interface{}) Message {
	return Message{
		ID:        id,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// Pure function to generate a unique ID
func GenerateUniqueID() string {
	// add uuid
	return uuid.New().String()
}

// Pure function to generate a random string
func RandomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[time.Now().UnixNano()%int64(len(letterBytes))]
		time.Sleep(1 * time.Nanosecond) // For uniqueness, but not pure due to side effect
	}
	return string(b)
}
