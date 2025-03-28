package pubsub

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/tphuc/pubsub/model"
)

// ReportData contains the data for a PubSub report
type ReportData struct {
	StartTime           time.Time      `json:"startTime"`
	EndTime             time.Time      `json:"endTime"`
	Duration            time.Duration  `json:"duration"`
	TotalMessages       int            `json:"totalMessages"`
	MessagesPerSecond   float64        `json:"messagesPerSecond"`
	ConsumerCount       int            `json:"consumerCount"`
	MessagesByConsumer  map[string]int `json:"messagesByConsumer"`
	MessageDistribution map[string]int `json:"messageDistribution"`
	ErrorCount          int            `json:"errorCount"`
	Errors              []string       `json:"errors"`
}

// Reporter generates reports about the PubSub system
type Reporter struct {
	mutex     sync.Mutex
	data      *ReportData
	startTime time.Time
	nextID    int
	errors    []string
	isActive  bool
}

// NewReporter creates a new reporter
func NewReporter() *Reporter {
	return &Reporter{
		data: &ReportData{
			MessagesByConsumer:  make(map[string]int),
			MessageDistribution: make(map[string]int),
			Errors:              make([]string, 0),
		},
		errors: make([]string, 0),
	}
}

// Start begins the reporting period
func (r *Reporter) Start() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.startTime = time.Now()
	r.data.StartTime = r.startTime
	r.isActive = true
}

// Stop ends the reporting period and finalizes the report
func (r *Reporter) Stop() *ReportData {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.isActive {
		return r.data
	}

	r.isActive = false
	endTime := time.Now()
	r.data.EndTime = endTime
	r.data.Duration = endTime.Sub(r.startTime)

	if r.data.Duration > 0 {
		r.data.MessagesPerSecond = float64(r.data.TotalMessages) / r.data.Duration.Seconds()
	}

	r.data.ConsumerCount = len(r.data.MessagesByConsumer)
	r.data.Errors = r.errors
	r.data.ErrorCount = len(r.errors)

	return r.data
}

// GetNextConsumerID generates a new unique ID for a consumer function
func (r *Reporter) GetNextConsumerID() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id := fmt.Sprintf("consumer-%d", r.nextID)
	r.nextID++
	return id
}

// RecordMessage records a message being processed
func (r *Reporter) RecordMessage(consumerID string, msg model.Message) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.isActive {
		return
	}

	// Increment total message count
	r.data.TotalMessages++

	// Track messages by consumer
	r.data.MessagesByConsumer[consumerID]++

	// Track message distribution by type
	msgType := fmt.Sprintf("%T", msg.Content)
	r.data.MessageDistribution[msgType]++
}

// RecordError records an error that occurred
func (r *Reporter) RecordError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.isActive {
		return
	}

	r.errors = append(r.errors, err.Error())
}

// WrapConsumer wraps a consumer function to record metrics
func (r *Reporter) WrapConsumer(consumer ConsumerFunc) ConsumerFunc {
	// Generate a unique ID for this consumer
	consumerID := r.GetNextConsumerID()

	return func(msg model.Message) {
		defer func() {
			if rec := recover(); rec != nil {
				err, ok := rec.(error)
				if !ok {
					err = fmt.Errorf("%v", rec)
				}
				r.RecordError(err)
			}
		}()

		r.RecordMessage(consumerID, msg)
		consumer(msg)
	}
}

// SaveReportToFile saves the report data to a file in JSON format
func (r *Reporter) SaveReportToFile(filename string) error {
	data := r.Stop() // Ensure the report is finalized

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// PrintReport prints the report to the specified writer (or stdout if nil)
func (r *Reporter) PrintReport(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}

	data := r.Stop() // Ensure the report is finalized

	fmt.Fprintf(w, "\nPubSub System Report\n")
	fmt.Fprintf(w, "====================\n\n")

	fmt.Fprintf(w, "Duration:             %v\n", data.Duration)
	fmt.Fprintf(w, "Total Messages:       %d\n", data.TotalMessages)
	fmt.Fprintf(w, "Messages Per Second:  %.2f\n", data.MessagesPerSecond)
	fmt.Fprintf(w, "Consumer Count:       %d\n", data.ConsumerCount)
	fmt.Fprintf(w, "Error Count:          %d\n", data.ErrorCount)

	fmt.Fprintf(w, "\nMessages by Consumer:\n")
	for consumerID, count := range data.MessagesByConsumer {
		fmt.Fprintf(w, "  %-20s %d\n", consumerID+":", count)
	}

	fmt.Fprintf(w, "\nMessage Distribution by Type:\n")
	for msgType, count := range data.MessageDistribution {
		fmt.Fprintf(w, "  %-30s %d\n", msgType+":", count)
	}

	if data.ErrorCount > 0 {
		fmt.Fprintf(w, "\nErrors:\n")
		for i, err := range data.Errors {
			fmt.Fprintf(w, "  %d. %s\n", i+1, err)
		}
	}
}

// Subscribe a consumer with reporting
func SubscribeWithReporting(ps *PubSub, reporter *Reporter, consumer ConsumerFunc) *PubSub {
	wrappedConsumer := reporter.WrapConsumer(consumer)
	return Subscribe(ps, wrappedConsumer)
}

// Method for backward compatibility
func (ps *PubSub) SubscribeWithReporting(reporter *Reporter, consumer ConsumerFunc) *PubSub {
	return SubscribeWithReporting(ps, reporter, consumer)
}
