package entites

import "time"

// Job represents a row in jobs. ID and DocumentID contain UUID strings.
// Nil pointers represent SQL NULL values.
type Job struct {
	ID         string     `db:"id" json:"id"`
	DocumentID string     `db:"document_id" json:"document_id"`
	Attempts   int32      `db:"attempts" json:"attempts"`
	LastError  *string    `db:"last_error" json:"last_error"`
	StartedAt  *time.Time `db:"started_at" json:"started_at"`
	FinishedAt *time.Time `db:"finished_at" json:"finished_at"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}
