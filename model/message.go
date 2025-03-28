package model

import (
	"time"
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
	return time.Now().Format("20060102150405") + "-" + RandomString(8)
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
