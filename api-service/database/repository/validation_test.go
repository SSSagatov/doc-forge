package repository

import (
	"encoding/json"
	"errors"
	"testing"

	"cloud-native-platform/api-service/entites"
	"github.com/jackc/pgx/v5"
)

func TestAnalysisResultJSONValidation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		data      json.RawMessage
		wantError bool
	}{
		{"SQL NULL", nil, false},
		{"JSON null", json.RawMessage(`null`), false},
		{"object", json.RawMessage(`{"key":1}`), false},
		{"array", json.RawMessage(`[1,"value"]`), false},
		{"empty", json.RawMessage{}, true},
		{"malformed", json.RawMessage(`{"key":`), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, value := range []entites.AnalysisResult{
				{Requirements: tc.data}, {Entities: tc.data}, {Risks: tc.data}, {RawResponse: tc.data},
			} {
				if err := validateAnalysisResult(&value); (err != nil) != tc.wantError {
					t.Fatalf("validation error = %v, want error %v", err, tc.wantError)
				}
			}
		})
	}
}

func TestCounterValidation(t *testing.T) {
	negative, zero := int32(-1), int32(0)
	if validateJob(&entites.Job{Attempts: negative}) == nil {
		t.Fatal("negative attempts accepted")
	}
	if err := validateJob(&entites.Job{}); err != nil {
		t.Fatal(err)
	}
	for _, value := range []entites.AnalysisResult{
		{PromptTokens: &negative}, {CompletionTokens: &negative},
	} {
		if validateAnalysisResult(&value) == nil {
			t.Fatal("negative tokens accepted")
		}
	}
	if err := validateAnalysisResult(&entites.AnalysisResult{PromptTokens: &zero}); err != nil {
		t.Fatal(err)
	}
	if validateJob(nil) == nil || validateAnalysisResult(nil) == nil {
		t.Fatal("nil input accepted")
	}
}

type errorRow struct{ err error }

func (r errorRow) Scan(...any) error { return r.err }

func TestScanErrors(t *testing.T) {
	if _, err := scanJob(errorRow{pgx.ErrNoRows}); !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("job error = %v", err)
	}
	if _, err := scanAnalysisResult(errorRow{pgx.ErrNoRows}); !errors.Is(err, ErrAnalysisResultNotFound) {
		t.Fatalf("result error = %v", err)
	}
	failure := errors.New("scan failed")
	if _, err := scanJob(errorRow{failure}); !errors.Is(err, failure) {
		t.Fatalf("job error = %v", err)
	}
	if _, err := scanAnalysisResult(errorRow{failure}); !errors.Is(err, failure) {
		t.Fatalf("result error = %v", err)
	}
}
