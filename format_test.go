package repogov_test

import (
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestPassed(t *testing.T) {
	tests := []struct {
		name    string
		results []repogov.Result
		want    bool
	}{
		{"nil results", nil, true},
		{"empty results", []repogov.Result{}, true},
		{"all pass", []repogov.Result{
			{Status: repogov.Pass},
			{Status: repogov.Pass},
		}, true},
		{"warn is passing", []repogov.Result{
			{Status: repogov.Pass},
			{Status: repogov.Warn},
		}, true},
		{"skip is passing", []repogov.Result{
			{Status: repogov.Skip},
		}, true},
		{"one fail", []repogov.Result{
			{Status: repogov.Pass},
			{Status: repogov.Fail},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repogov.Passed(tt.results); got != tt.want {
				t.Errorf("Passed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := repogov.Summary(nil)
		if got != "No files checked." {
			t.Errorf("Summary(nil) = %q, want 'No files checked.'", got)
		}
	})

	t.Run("mixed results", func(t *testing.T) {
		results := []repogov.Result{
			{Path: "a.md", Lines: 10, Limit: 500, Status: repogov.Pass},
			{Path: "b.md", Lines: 450, Limit: 500, Status: repogov.Warn},
			{Path: "c.md", Lines: 600, Limit: 500, Status: repogov.Fail},
		}
		got := repogov.Summary(results)
		if !strings.Contains(got, "1 pass") {
			t.Error("expected '1 pass' in summary")
		}
		if !strings.Contains(got, "1 warn") {
			t.Error("expected '1 warn' in summary")
		}
		if !strings.Contains(got, "1 fail") {
			t.Error("expected '1 fail' in summary")
		}
		if !strings.Contains(got, "3 files") {
			t.Error("expected '3 files' in summary")
		}
	})
}

func TestLayoutPassed(t *testing.T) {
	tests := []struct {
		name    string
		results []repogov.LayoutResult
		want    bool
	}{
		{"nil", nil, true},
		{"all pass", []repogov.LayoutResult{
			{Status: repogov.Pass},
			{Status: repogov.Info},
		}, true},
		{"one fail", []repogov.LayoutResult{
			{Status: repogov.Pass},
			{Status: repogov.Fail},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repogov.LayoutPassed(tt.results); got != tt.want {
				t.Errorf("LayoutPassed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLayoutSummary(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := repogov.LayoutSummary(nil)
		if got != "No layout checks performed." {
			t.Errorf("LayoutSummary(nil) = %q", got)
		}
	})

	t.Run("with results", func(t *testing.T) {
		results := []repogov.LayoutResult{
			{Path: ".github/workflows/ci.yml", Status: repogov.Pass, Message: "required file present"},
			{Path: ".github/CODEOWNERS", Status: repogov.Info, Message: "optional file present"},
		}
		got := repogov.LayoutSummary(results)
		if !strings.Contains(got, "1 pass") {
			t.Error("expected '1 pass' in layout summary")
		}
		if !strings.Contains(got, "1 info") {
			t.Error("expected '1 info' in layout summary")
		}
	})
}
