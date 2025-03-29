package model

import (
	"strings"
	"testing"
)

func TestNewMessage(t *testing.T) {
	content := "test content"
	msg := NewMessage(content)

	if msg.Content != content {
		t.Errorf("Expected content %v, got %v", content, msg.Content)
	}
	if msg.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if msg.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestNewMessageWithID(t *testing.T) {
	id := "test-id"
	content := "test content"
	msg := NewMessageWithID(id, content)

	if msg.ID != id {
		t.Errorf("Expected ID %s, got %s", id, msg.ID)
	}
	if msg.Content != content {
		t.Errorf("Expected content %v, got %v", content, msg.Content)
	}
	if msg.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestGenerateUniqueID(t *testing.T) {
	// Generate multiple IDs and ensure they're unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateUniqueID()
		if ids[id] {
			t.Error("Generated duplicate ID:", id)
		}
		ids[id] = true

		// Verify UUID format
		if len(strings.Split(id, "-")) != 5 {
			t.Error("Invalid UUID format:", id)
		}
	}
}

func TestRandomString(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"empty string", 0},
		{"short string", 5},
		{"medium string", 10},
		{"long string", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RandomString(tt.n)
			if len(result) != tt.n {
				t.Errorf("Expected length %d, got %d", tt.n, len(result))
			}

			// Check if string contains only valid characters
			validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			for _, char := range result {
				if !strings.ContainsRune(validChars, char) {
					t.Errorf("Invalid character in random string: %c", char)
				}
			}
		})
	}

	// Test that strings are of correct length
	str := RandomString(10)
	if len(str) != 10 {
		t.Errorf("Expected length 10, got %d", len(str))
	}

	// Test that strings contain only valid characters
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, char := range str {
		if !strings.ContainsRune(validChars, char) {
			t.Errorf("Invalid character in random string: %c", char)
		}
	}
}
