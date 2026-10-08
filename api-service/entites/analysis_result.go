package entites

import (
	"encoding/json"
	"time"
)

// AnalysisResult represents a row in analysis_results.
// ID and JobID contain UUID strings. Nil pointers and nil RawMessage values
// represent SQL NULL; RawMessage preserves the unrestricted JSONB structure.
type AnalysisResult struct {
	ID               string          `db:"id" json:"id"`
	JobID            string          `db:"job_id" json:"job_id"`
	Summary          *string         `db:"summary" json:"summary"`
	Requirements     json.RawMessage `db:"requirements" json:"requirements"`
	Entities         json.RawMessage `db:"entities" json:"entities"`
	Risks            json.RawMessage `db:"risks" json:"risks"`
	RawResponse      json.RawMessage `db:"raw_response" json:"raw_response"`
	PromptTokens     *int32          `db:"prompt_tokens" json:"prompt_tokens"`
	CompletionTokens *int32          `db:"completion_tokens" json:"completion_tokens"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
}
