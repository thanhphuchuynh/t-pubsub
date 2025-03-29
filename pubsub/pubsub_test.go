package pubsub

import (
	"testing"
	"time"

	"github.com/thanhphuchuynh/t-pubsub/model"
)

func TestNewPubSub(t *testing.T) {
	ps := NewPubSub()
	if ps == nil {
		t.Fatal("Expected non-nil PubSub instance")
	}
	if ps.store == nil {
		t.Error("Expected non-nil message store")
	}
	if len(ps.consumers) != 0 {
		t.Errorf("Expected 0 consumers, got %d", len(ps.consumers))
	}
}

func TestPublish(t *testing.T) {
	ps := NewPubSub()

	tests := []struct {
		name    string
		content interface{}
	}{
		{"string message", "test message"},
		{"int message", 42},
		{"map message", map[string]string{"key": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgID := ps.Publish(tt.content)
			if msgID == "" {
				t.Error("Expected non-empty message ID")
			}

			// Verify message was stored
			msg, found := ps.store.GetMessageByID(msgID)
			if !found {
				t.Error("Message not found in store")
			}

			switch v := tt.content.(type) {
			case string:
				if msg.Content != v {
					t.Errorf("Expected content %v, got %v", v, msg.Content)
				}
			case int:
				if msg.Content != v {
					t.Errorf("Expected content %v, got %v", v, msg.Content)
				}
			case map[string]string:
				content, ok := msg.Content.(map[string]string)
				if !ok {
					t.Fatalf("Expected content to be map[string]string, got %T", msg.Content)
				}
				if len(content) != len(v) {
					t.Fatalf("Expected map length %d, got %d", len(v), len(content))
				}
				for key, val := range v {
					if content[key] != val {
						t.Errorf("Expected map[%s] = %s, got %s", key, val, content[key])
					}
				}
			default:
				t.Fatalf("Unexpected type %T", tt.content)
			}
		})
	}
}

func TestSubscribe(t *testing.T) {
	ps := NewPubSub()
	received := make(chan model.Message, 1)

	consumer := func(msg model.Message) {
		received <- msg
	}

	ps = ps.Subscribe(consumer)
	if len(ps.GetConsumers()) != 1 {
		t.Errorf("Expected 1 consumer, got %d", len(ps.GetConsumers()))
	}

	// Test message delivery
	content := "test message"
	msgID := ps.Publish(content)
	ps.PushByID(msgID)

	select {
	case msg := <-received:
		if msg.Content != content {
			t.Errorf("Expected content %v, got %v", content, msg.Content)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestMessageStore(t *testing.T) {
	store := NewInMemoryMessageStore()

	t.Run("add and get message", func(t *testing.T) {
		msg := model.NewMessage("test")
		store.AddMessage(msg)

		messages := store.GetMessages()
		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}
		if messages[0].Content != "test" {
			t.Errorf("Expected content 'test', got %v", messages[0].Content)
		}
	})

	t.Run("get by ID", func(t *testing.T) {
		msg := model.NewMessageWithID("test-id", "test content")
		store.AddMessage(msg)

		found, exists := store.GetMessageByID("test-id")
		if !exists {
			t.Error("Message not found")
		}
		if found.Content != "test content" {
			t.Errorf("Expected content 'test content', got %v", found.Content)
		}
	})

	t.Run("clear messages", func(t *testing.T) {
		store.AddMessage(model.NewMessage("test1"))
		store.AddMessage(model.NewMessage("test2"))

		store.ClearMessages()
		messages := store.GetMessages()
		if len(messages) != 0 {
			t.Errorf("Expected 0 messages after clear, got %d", len(messages))
		}
	})
}

func TestPushOperations(t *testing.T) {
	ps := NewPubSub()
	received := make(chan model.Message, 10)

	consumer := func(msg model.Message) {
		received <- msg
	}

	ps = ps.Subscribe(consumer)

	t.Run("push by ID", func(t *testing.T) {
		msgID := ps.Publish("test1")
		success := ps.PushByID(msgID)

		if !success {
			t.Error("PushByID returned false")
		}

		select {
		case msg := <-received:
			if msg.Content != "test1" {
				t.Errorf("Expected content 'test1', got %v", msg.Content)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for message")
		}
	})

	t.Run("push all", func(t *testing.T) {
		// Clear previous messages
		ps.Clear()

		// Publish multiple messages
		ps.Publish("test2")
		ps.Publish("test3")

		ps.PushAll()

		// Should receive both messages
		receivedCount := 0
		timeout := time.After(time.Second)

		for receivedCount < 2 {
			select {
			case <-received:
				receivedCount++
			case <-timeout:
				t.Errorf("Timeout waiting for messages. Got %d of 2", receivedCount)
				return
			}
		}
	})
}

func TestAutoPush(t *testing.T) {
	ps := NewPubSub(WithPushInterval(50 * time.Millisecond))
	received := make(chan model.Message, 10)

	consumer := func(msg model.Message) {
		received <- msg
	}

	ps = ps.Subscribe(consumer)
	ps = ps.StartAutoPush()

	// Publish a message
	ps.Publish("auto test")

	// Wait for auto push
	select {
	case msg := <-received:
		if msg.Content != "auto test" {
			t.Errorf("Expected content 'auto test', got %v", msg.Content)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for auto push")
	}

	ps = ps.StopAutoPush()
}
