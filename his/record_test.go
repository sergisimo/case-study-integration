package his

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"discharge-bridge/discharge"
)

func TestFetcherFetch(t *testing.T) {
	fixture, err := os.ReadFile("../testdata/discharge_records.json")
	if err != nil {
		t.Fatalf("read discharge records fixture: %v", err)
	}

	const apiKey = "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("request method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/records" {
			t.Errorf("request path = %q, want %q", r.URL.Path, "/records")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+apiKey {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer "+apiKey)
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(fixture); err != nil {
			t.Errorf("write fixture response: %v", err)
		}
	}))
	defer server.Close()

	got, err := NewFetcher(server.URL, apiKey).Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	want := []discharge.Summary{
		{
			PatientID:     "H-10293",
			PatientName:   "Anna Musterfrau",
			DOB:           time.Date(1957, time.March, 14, 0, 0, 0, 0, time.UTC),
			InsuranceNr:   "DE-AOK-88214477",
			DischargeDate: time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC),
			Conditions:    []string{"I50.9"},
			WeightKg:      68.5,
			Notes:         "Routine discharge, no complications.",
		},
		{
			PatientID:     "H-10294",
			PatientName:   "Klaus Beispiel",
			DOB:           time.Date(1942, time.November, 2, 0, 0, 0, 0, time.UTC),
			InsuranceNr:   "DE-TK-55021198",
			DischargeDate: time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC),
			Conditions:    []string{"I50.9", "E11.9", "N18.3"},
			WeightKg:      79.83,
			Notes:         "Callback requested at 0151-2345678, home address Musterstr. 5, 12345 Berlin.",
		},
		{
			PatientID:     "H-10295",
			PatientName:   "Petra Beispielhaft",
			DOB:           time.Date(1989, time.June, 30, 0, 0, 0, 0, time.UTC),
			InsuranceNr:   "DE-BARMER-33210099",
			DischargeDate: time.Date(2026, time.September, 19, 0, 0, 0, 0, time.UTC),
			Conditions:    []string{},
		},
		{
			PatientID:     "H-10296",
			PatientName:   "Otto Testmann",
			DOB:           time.Date(1975, time.January, 22, 0, 0, 0, 0, time.UTC),
			InsuranceNr:   "DE-DAK-77341205",
			DischargeDate: time.Date(2026, time.September, 19, 0, 0, 0, 0, time.UTC),
			Conditions:    []string{"J44.9", "I10"},
			WeightKg:      89.81,
			Notes:         "Next of kin: Maria Testmann, reachable at maria.testmann@example.com.",
		},
		{
			PatientID:     "H-10297",
			PatientName:   "Sabine Musterhausen",
			DOB:           time.Date(1963, time.August, 9, 0, 0, 0, 0, time.UTC),
			InsuranceNr:   "DE-AOK-90112233",
			DischargeDate: time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC),
			Conditions:    []string{"E11.9"},
			WeightKg:      81.2,
			Notes:         "Patient requested records not be shared with employer.",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Fetch() summaries differ:\n got: %#v\nwant: %#v", got, want)
	}
}
