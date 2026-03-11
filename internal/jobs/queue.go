package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

const TypeEnhanceAchievement = "ai_enhance"
const TypeMapKPI = "kpi_mapping"

type Job struct {
	ID        int64
	UserID    int64
	Type      string
	Status    string
	Payload   []byte
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Queue struct {
	db *sql.DB
}

func NewQueue(db *sql.DB) *Queue {
	return &Queue{db: db}
}

func (q *Queue) Enqueue(ctx context.Context, userID int64, jobType string, payload []byte) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO jobs (user_id, type, payload)
		VALUES (?, ?, ?)
	`, userID, jobType, string(payload))
	return err
}

func (q *Queue) ClaimPending(ctx context.Context) (*Job, error) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var job Job
	err = tx.QueryRowContext(ctx, `
		SELECT id, user_id, type, status, payload, error, created_at, updated_at
		FROM jobs
		WHERE status = 'pending'
		ORDER BY created_at
		LIMIT 1
	`).Scan(&job.ID, &job.UserID, &job.Type, &job.Status, &job.Payload, &job.Error, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE jobs
		SET status = 'processing', updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, job.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	job.Status = "processing"
	return &job, nil
}

func (q *Queue) Complete(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE jobs SET status = 'completed', error = '', updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, id)
	return err
}

func (q *Queue) Fail(ctx context.Context, id int64, jobErr error) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE jobs SET status = 'failed', error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, jobErr.Error(), id)
	return err
}

func DecodePayload[T any](payload []byte) (T, error) {
	var value T
	if err := json.Unmarshal(payload, &value); err != nil {
		return value, fmt.Errorf("decode payload: %w", err)
	}
	return value, nil
}
