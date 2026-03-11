package achievements

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/kpi-chaser/internal/jobs"
)

type Achievement struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"userId"`
	Quarter      string     `json:"quarter"`
	RawText      string     `json:"rawText"`
	EnhancedText string     `json:"enhancedText"`
	Category     string     `json:"category"`
	ImpactNote   string     `json:"impactNote"`
	KPIID        *int64     `json:"kpiId"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type Service struct {
	db    *sql.DB
	queue *jobs.Queue
}

func NewService(db *sql.DB, queue *jobs.Queue) *Service {
	return &Service{db: db, queue: queue}
}

func (s *Service) List(ctx context.Context, userID int64, quarter string) ([]Achievement, error) {
	query := `
		SELECT id, user_id, quarter, raw_text, enhanced_text, category, impact_note, kpi_id, status, created_at, updated_at
		FROM achievements
		WHERE user_id = ?
	`
	args := []any{userID}
	if quarter != "" {
		query += ` AND quarter = ?`
		args = append(args, quarter)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list achievements: %w", err)
	}
	defer rows.Close()

	var items []Achievement
	for rows.Next() {
		var item Achievement
		if err := rows.Scan(&item.ID, &item.UserID, &item.Quarter, &item.RawText, &item.EnhancedText, &item.Category, &item.ImpactNote, &item.KPIID, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) Create(ctx context.Context, input Achievement) (Achievement, error) {
	if strings.TrimSpace(input.RawText) == "" {
		return Achievement{}, errors.New("raw text is required")
	}
	if strings.TrimSpace(input.Quarter) == "" {
		return Achievement{}, errors.New("quarter is required")
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO achievements (user_id, quarter, raw_text, enhanced_text, category, impact_note, kpi_id, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, input.UserID, input.Quarter, input.RawText, input.EnhancedText, input.Category, input.ImpactNote, input.KPIID, "draft")
	if err != nil {
		return Achievement{}, fmt.Errorf("create achievement: %w", err)
	}
	id, _ := result.LastInsertId()
	return s.Get(ctx, input.UserID, id)
}

func (s *Service) Get(ctx context.Context, userID, id int64) (Achievement, error) {
	var item Achievement
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, quarter, raw_text, enhanced_text, category, impact_note, kpi_id, status, created_at, updated_at
		FROM achievements
		WHERE user_id = ? AND id = ?
	`, userID, id).Scan(&item.ID, &item.UserID, &item.Quarter, &item.RawText, &item.EnhancedText, &item.Category, &item.ImpactNote, &item.KPIID, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Service) EnqueueEnhancement(ctx context.Context, userID, achievementID int64) error {
	payload, err := json.Marshal(map[string]int64{"achievementId": achievementID})
	if err != nil {
		return err
	}
	return s.queue.Enqueue(ctx, userID, jobs.TypeEnhanceAchievement, payload)
}

func (s *Service) Update(ctx context.Context, input Achievement) (Achievement, error) {
	if strings.TrimSpace(input.Quarter) == "" {
		return Achievement{}, errors.New("quarter is required")
	}
	if strings.TrimSpace(input.RawText) == "" {
		return Achievement{}, errors.New("raw text is required")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE achievements
		SET quarter = ?, raw_text = ?, enhanced_text = ?, category = ?, impact_note = ?, kpi_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`, input.Quarter, input.RawText, input.EnhancedText, input.Category, input.ImpactNote, input.KPIID, input.ID, input.UserID)
	if err != nil {
		return Achievement{}, fmt.Errorf("update achievement: %w", err)
	}
	return s.Get(ctx, input.UserID, input.ID)
}

func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM achievements WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete achievement: %w", err)
	}
	return nil
}
