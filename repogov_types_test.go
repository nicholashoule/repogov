package repogov_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		status repogov.Status
		want   string
	}{
		{repogov.Pass, "PASS"},
		{repogov.Warn, "WARN"},
		{repogov.Fail, "FAIL"},
		{repogov.Skip, "SKIP"},
		{repogov.Info, "INFO"},
		{repogov.Status(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Status(%d).String() = %q, want %q", int(tt.status), got, tt.want)
			}
		})
	}
}

func TestStatusMarshalJSON(t *testing.T) {
	tests := []struct {
		status repogov.Status
		want   string
	}{
		{repogov.Pass, `"PASS"`},
		{repogov.Warn, `"WARN"`},
		{repogov.Fail, `"FAIL"`},
		{repogov.Skip, `"SKIP"`},
		{repogov.Info, `"INFO"`},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.want {
				t.Errorf("MarshalJSON = %s, want %s", data, tt.want)
			}
		})
	}
}

func TestStatusUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  repogov.Status
	}{
		{`"PASS"`, repogov.Pass},
		{`"WARN"`, repogov.Warn},
		{`"FAIL"`, repogov.Fail},
		{`"SKIP"`, repogov.Skip},
		{`"INFO"`, repogov.Info},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var s repogov.Status
			if err := json.Unmarshal([]byte(tt.input), &s); err != nil {
				t.Fatal(err)
			}
			if s != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v", tt.input, s, tt.want)
			}
		})
	}
}

func TestStatusUnmarshalJSON_Invalid(t *testing.T) {
	var s repogov.Status
	if err := json.Unmarshal([]byte(`"BOGUS"`), &s); err == nil {
		t.Error("expected error for unknown status label")
	}
}

func TestResultJSON_RoundTrip(t *testing.T) {
	r := repogov.Result{
		Path:   "test.go",
		Lines:  42,
		Limit:  500,
		Status: repogov.Pass,
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	// Verify Status appears as string, not integer.
	if got := string(data); !strings.Contains(got, `"Status":"PASS"`) {
		t.Errorf("JSON should contain Status string, got: %s", got)
	}
	var r2 repogov.Result
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatal(err)
	}
	if r2.Status != repogov.Pass {
		t.Errorf("round-trip Status = %v, want PASS", r2.Status)
	}
}
