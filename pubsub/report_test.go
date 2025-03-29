package pubsub

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/thanhphuchuynh/t-pubsub/model"
)

func TestReporter(t *testing.T) {
	t.Run("basic reporter functionality", func(t *testing.T) {
		reporter := NewReporter()
		if reporter == nil {
			t.Fatal("Expected non-nil reporter")
		}

		reporter.Start()
		if !reporter.isActive {
			t.Error("Expected reporter to be active after Start()")
		}

		data := reporter.Stop()
		if reporter.isActive {
			t.Error("Expected reporter to be inactive after Stop()")
		}

		if data.StartTime.IsZero() {
			t.Error("Expected non-zero start time")
		}
		if data.EndTime.IsZero() {
			t.Error("Expected non-zero end time")
		}
	})

	t.Run("consumer ID generation", func(t *testing.T) {
		reporter := NewReporter()
		id1 := reporter.GetNextConsumerID()
		id2 := reporter.GetNextConsumerID()

		if id1 == id2 {
			t.Error("Expected unique consumer IDs")
		}
		if id1 != "consumer-0" {
			t.Errorf("Expected first ID to be 'consumer-0', got %s", id1)
		}
		if id2 != "consumer-1" {
			t.Errorf("Expected second ID to be 'consumer-1', got %s", id2)
		}
	})

	t.Run("message recording", func(t *testing.T) {
		reporter := NewReporter()
		reporter.Start()

		msg1 := model.NewMessage("test1")
		msg2 := model.NewMessage(42)

		reporter.RecordMessage("consumer-1", msg1)
		reporter.RecordMessage("consumer-1", msg2)
		reporter.RecordMessage("consumer-2", msg1)

		data := reporter.Stop()

		if data.TotalMessages != 3 {
			t.Errorf("Expected 3 total messages, got %d", data.TotalMessages)
		}
		if data.ConsumerCount != 2 {
			t.Errorf("Expected 2 consumers, got %d", data.ConsumerCount)
		}
		if data.MessagesByConsumer["consumer-1"] != 2 {
			t.Errorf("Expected 2 messages for consumer-1, got %d", data.MessagesByConsumer["consumer-1"])
		}
		if data.MessagesByConsumer["consumer-2"] != 1 {
			t.Errorf("Expected 1 message for consumer-2, got %d", data.MessagesByConsumer["consumer-2"])
		}

		// Check message distribution
		stringType := "string"
		intType := "int"
		if data.MessageDistribution[stringType] != 2 {
			t.Errorf("Expected 2 string messages, got %d", data.MessageDistribution[stringType])
		}
		if data.MessageDistribution[intType] != 1 {
			t.Errorf("Expected 1 int message, got %d", data.MessageDistribution[intType])
		}
	})

	t.Run("error recording", func(t *testing.T) {
		reporter := NewReporter()
		reporter.Start()

		err1 := errors.New("test error 1")
		err2 := errors.New("test error 2")
		reporter.RecordError(err1)
		reporter.RecordError(err2)

		data := reporter.Stop()

		if data.ErrorCount != 2 {
			t.Errorf("Expected 2 errors, got %d", data.ErrorCount)
		}
		if data.Errors[0] != err1.Error() {
			t.Errorf("Expected first error to be '%s', got '%s'", err1.Error(), data.Errors[0])
		}
		if data.Errors[1] != err2.Error() {
			t.Errorf("Expected second error to be '%s', got '%s'", err2.Error(), data.Errors[1])
		}
	})

	t.Run("consumer wrapping", func(t *testing.T) {
		reporter := NewReporter()
		reporter.Start()

		messages := make(chan model.Message, 1)
		consumer := func(msg model.Message) {
			messages <- msg
		}

		wrappedConsumer := reporter.WrapConsumer(consumer)
		testMsg := model.NewMessage("test")
		wrappedConsumer(testMsg)

		select {
		case received := <-messages:
			if received.Content != "test" {
				t.Errorf("Expected message content 'test', got '%v'", received.Content)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for message")
		}

		data := reporter.Stop()
		if data.TotalMessages != 1 {
			t.Error("Expected wrapped consumer to record message")
		}
	})

	t.Run("panic handling in wrapped consumer", func(t *testing.T) {
		reporter := NewReporter()
		reporter.Start()

		consumer := func(msg model.Message) {
			panic("test panic")
		}

		wrappedConsumer := reporter.WrapConsumer(consumer)
		testMsg := model.NewMessage("test")

		// The panic should be caught
		wrappedConsumer(testMsg)

		data := reporter.Stop()
		if data.ErrorCount != 1 {
			t.Error("Expected panic to be recorded as error")
		}
		if data.Errors[0] != "test panic" {
			t.Errorf("Expected error message 'test panic', got '%s'", data.Errors[0])
		}
	})

	t.Run("report file saving", func(t *testing.T) {
		reporter := NewReporter()
		reporter.Start()
		reporter.RecordMessage("consumer-1", model.NewMessage("test"))

		tempFile := "test_report.json"
		defer os.Remove(tempFile)

		err := reporter.SaveReportToFile(tempFile)
		if err != nil {
			t.Fatalf("Failed to save report: %v", err)
		}

		// Read and verify the saved file
		content, err := os.ReadFile(tempFile)
		if err != nil {
			t.Fatalf("Failed to read report file: %v", err)
		}

		var savedData ReportData
		if err := json.Unmarshal(content, &savedData); err != nil {
			t.Fatalf("Failed to parse saved report: %v", err)
		}

		if savedData.TotalMessages != 1 {
			t.Errorf("Expected 1 message in saved report, got %d", savedData.TotalMessages)
		}
	})

	t.Run("report printing", func(t *testing.T) {
		reporter := NewReporter()
		reporter.Start()
		reporter.RecordMessage("consumer-1", model.NewMessage("test"))
		reporter.RecordError(errors.New("test error"))

		var buf bytes.Buffer
		reporter.PrintReport(&buf)

		output := buf.String()
		if len(output) == 0 {
			t.Error("Expected non-empty report output")
		}
		if !bytes.Contains(buf.Bytes(), []byte("PubSub System Report")) {
			t.Error("Expected report header in output")
		}
		if !bytes.Contains(buf.Bytes(), []byte("Total Messages:       1")) {
			t.Error("Expected message count in output")
		}
		if !bytes.Contains(buf.Bytes(), []byte("test error")) {
			t.Error("Expected error message in output")
		}
	})

	t.Run("integration with PubSub", func(t *testing.T) {
		ps := NewPubSub()
		reporter := NewReporter()
		reporter.Start()

		messages := make(chan model.Message, 1)
		consumer := func(msg model.Message) {
			messages <- msg
		}

		ps = ps.SubscribeWithReporting(reporter, consumer)
		msgID := ps.Publish("test")
		ps.PushByID(msgID)

		select {
		case received := <-messages:
			if received.Content != "test" {
				t.Errorf("Expected message content 'test', got '%v'", received.Content)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for message")
		}

		data := reporter.Stop()
		if data.TotalMessages != 1 {
			t.Error("Expected message to be recorded in report")
		}
	})
}
