package his

import (
	"context"
	"discharge-bridge/discharge"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

type (
	Fetcher interface {
		Fetch(context.Context) ([]discharge.Summary, error)
	}

	record struct {
		PatientID     string  `json:"PatientID"`
		PatientName   string  `json:"PatientName,omitempty"`
		DOB           string  `json:"DOB"`
		InsuranceNr   string  `json:"InsuranceNr"`
		DischargeDate string  `json:"DischargeDate"`
		Diagnosis     string  `json:"Diagnosis"` // comma-separated ICD-10 codes, or empty
		WeightKg      float64 `json:"WeightKg"`  // 0 if not recorded in kg
		WeightLbs     float64 `json:"WeightLbs"` // 0 if not recorded in lbs
		Notes         string  `json:"Notes"`
	}

	hisFetcher struct {
		url    string
		apiKey string
	}
)

func NewFetcher(url, apiKey string) *hisFetcher {
	return &hisFetcher{
		url:    url,
		apiKey: apiKey,
	}
}

func (f *hisFetcher) Fetch(ctx context.Context) ([]discharge.Summary, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", f.url+"/records", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HIS request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+f.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch records from HIS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("HIS returned %d and failed to read body: %v", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("HIS returned %d: %s", resp.StatusCode, string(body))
	}

	var records []record
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, fmt.Errorf("failed to parse HIS response: %v", err)
	}

	summaries := make([]discharge.Summary, len(records))
	for i, rec := range records {
		summaries[i], err = transform(rec)
		if err != nil {
			return nil, fmt.Errorf("failed to transform record: %v", err)
		}
	}
	return summaries, nil
}

func transform(rec record) (discharge.Summary, error) {
	weightKg := rec.WeightKg
	if weightKg == 0 && rec.WeightLbs != 0 {
		weightKg = lbsToKg(rec.WeightLbs)
	}

	conditions := []string{}
	if diagnosis := strings.TrimSpace(rec.Diagnosis); diagnosis != "" {
		for _, condition := range strings.Split(diagnosis, ",") {
			conditions = append(conditions, strings.TrimSpace(condition))
		}
	}

	dob, err := time.Parse("2006-01-02", rec.DOB)
	if err != nil {
		return discharge.Summary{}, err
	}

	dischargeDate, err := time.Parse("2006-01-02", rec.DischargeDate)
	if err != nil {
		return discharge.Summary{}, err
	}

	return discharge.Summary{
		PatientID:     rec.PatientID,
		PatientName:   rec.PatientName,
		DOB:           dob,
		InsuranceNr:   rec.InsuranceNr,
		DischargeDate: dischargeDate,
		Conditions:    conditions,
		WeightKg:      weightKg,
		Notes:         rec.Notes,
	}, nil
}

// lbsToKg converts pounds to kilograms, truncating to two decimal places.
func lbsToKg(lbs float64) float64 {
	const lbsToKgFactor = 0.45359237
	return math.Trunc(lbs*lbsToKgFactor*100) / 100
}
