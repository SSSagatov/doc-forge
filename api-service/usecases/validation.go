package usecases

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"cloud-native-platform/api-service/entites"
)

// ErrInvalidInput can be detected with errors.Is.
var ErrInvalidInput = errors.New("invalid input")

func invalid(message string) error { return fmt.Errorf("%w: %s", ErrInvalidInput, message) }

// IDs use the canonical UUID spelling: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func validateID(id string) error {
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return invalid("ID must be a canonical UUID")
	}
	compact := id[:8] + id[9:13] + id[14:18] + id[19:23] + id[24:]
	if _, err := hex.DecodeString(compact); err != nil {
		return invalid("ID must be a canonical UUID")
	}
	return nil
}

func validatePagination(limit, offset int) error {
	if limit < 1 || limit > 100 || offset < 0 {
		return invalid("limit must be 1..100 and offset must be non-negative")
	}
	return nil
}

func validStatus(status entites.DocumentStatus) bool {
	switch status {
	case entites.DocumentStatusUploaded, entites.DocumentStatusProcessing, entites.DocumentStatusCompleted, entites.DocumentStatusFailed:
		return true
	default:
		return false
	}
}

func validateDocument(value *entites.Document) error {
	if value == nil {
		return invalid("document is nil")
	}
	if strings.TrimSpace(value.OriginalFilename) == "" || strings.TrimSpace(value.StorageKey) == "" || strings.TrimSpace(value.MIMEType) == "" {
		return invalid("filename, storage key and MIME type are required")
	}
	if value.SizeBytes < 0 {
		return invalid("size must not be negative")
	}
	if value.Status != "" && !validStatus(value.Status) {
		return invalid("unknown document status")
	}
	return nil
}

func validateJob(value *entites.Job) error {
	if value == nil {
		return invalid("job is nil")
	}
	if err := validateID(value.DocumentID); err != nil {
		return fmt.Errorf("document ID: %w", err)
	}
	if value.Attempts < 0 {
		return invalid("attempts must not be negative")
	}
	if value.StartedAt != nil && value.FinishedAt != nil && value.FinishedAt.Before(*value.StartedAt) {
		return invalid("finish time must not precede start time")
	}
	return nil
}

func validateAnalysisResult(value *entites.AnalysisResult) error {
	if value == nil {
		return invalid("analysis result is nil")
	}
	if err := validateID(value.JobID); err != nil {
		return fmt.Errorf("job ID: %w", err)
	}
	if value.PromptTokens != nil && *value.PromptTokens < 0 || value.CompletionTokens != nil && *value.CompletionTokens < 0 {
		return invalid("token counts must not be negative")
	}
	for _, field := range []struct {
		name string
		data json.RawMessage
	}{
		{"requirements", value.Requirements}, {"entities", value.Entities}, {"risks", value.Risks}, {"raw_response", value.RawResponse},
	} {
		if field.data != nil && !json.Valid(field.data) {
			return invalid(field.name + " must contain valid JSON or be nil")
		}
	}
	return nil
}
