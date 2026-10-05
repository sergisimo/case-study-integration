package discharge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type (
	Client interface {
		Send(ctx context.Context, summary Summary) error
	}

	Summary struct {
		PatientID     string    `json:"patientId"`
		PatientName   string    `json:"patientName"`
		DOB           time.Time `json:"dob"`
		InsuranceNr   string    `json:"insuranceNr"`
		DischargeDate time.Time `json:"dischargeDate"`
		Conditions    []string  `json:"conditions"`
		WeightKg      float64   `json:"weightKg"`
		Notes         string    `json:"notes"`
	}

	client struct {
		url    string
		apiKey string
	}
)

func (s Summary) MarshalRedactedJSON() ([]byte, error) {
	s.PatientName = ""
	return json.Marshal(s)
}

func NewClient(url, apiKey string) *client {
	return &client{
		url:    url,
		apiKey: apiKey,
	}
}

func (c *client) Send(ctx context.Context, summary Summary) error {
	maxRetries := 5
	backoff := 500 * time.Millisecond

	for {
		err := c.sendSummary(ctx, summary)
		if err != nil {
			if maxRetries == 0 {
				c.logFailedRecord(summary)
				log.Println("failed to send record", summary.PatientID)
				return err
			}
			maxRetries--
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		log.Println("successfully sent record", summary.PatientID)
		return nil
	}
}

func (c *client) sendSummary(ctx context.Context, summary Summary) error {
	payload, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("failed to marshal discharge summary: %v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", c.url+"/discharge-summaries", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create Recare request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("recare returned %d and failed to read body: %v", resp.StatusCode, err)
		}
		return fmt.Errorf("recare returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// logFailedRecord keeps a local copy of anything that failed to send so we
// can retry or inspect it later.
func (c *client) logFailedRecord(sum Summary) {
	f, err := os.OpenFile("failed_records.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open log file: %v\n", err)
		return
	}
	defer f.Close()

	raw, err := sum.MarshalRedactedJSON()
	if err != nil {
		log.Printf("failed to marshal redacted JSON for record %s: %v\n", sum.PatientID, err)
		return
	}

	_, err = f.Write(append(raw, '\n'))
	if err != nil {
		log.Printf("failed to write record %s to log file: %v\n", sum.PatientID, err)
	}
}
