package kpi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type KPI struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"userId"`
	Quarter      string    `json:"quarter"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Weight       int       `json:"weight"`
	TargetMetric string    `json:"targetMetric"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, userID int64, quarter string) ([]KPI, error) {
	query := `
		SELECT id, user_id, quarter, title, description, weight, target_metric, created_at, updated_at
		FROM kpis
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
		return nil, fmt.Errorf("list kpis: %w", err)
	}
	defer rows.Close()

	var kpis []KPI
	for rows.Next() {
		var item KPI
		if err := rows.Scan(&item.ID, &item.UserID, &item.Quarter, &item.Title, &item.Description, &item.Weight, &item.TargetMetric, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		kpis = append(kpis, item)
	}
	return kpis, rows.Err()
}

func (s *Service) Create(ctx context.Context, input KPI) (KPI, error) {
	if strings.TrimSpace(input.Quarter) == "" {
		return KPI{}, errors.New("quarter is required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return KPI{}, errors.New("title is required")
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO kpis (user_id, quarter, title, description, weight, target_metric)
		VALUES (?, ?, ?, ?, ?, ?)
	`, input.UserID, input.Quarter, input.Title, input.Description, input.Weight, input.TargetMetric)
	if err != nil {
		return KPI{}, fmt.Errorf("create kpi: %w", err)
	}

	id, _ := result.LastInsertId()
	return s.Get(ctx, input.UserID, id)
}

func (s *Service) Get(ctx context.Context, userID, id int64) (KPI, error) {
	var item KPI
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, quarter, title, description, weight, target_metric, created_at, updated_at
		FROM kpis
		WHERE user_id = ? AND id = ?
	`, userID, id).Scan(&item.ID, &item.UserID, &item.Quarter, &item.Title, &item.Description, &item.Weight, &item.TargetMetric, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Service) Update(ctx context.Context, input KPI) (KPI, error) {
	if strings.TrimSpace(input.Quarter) == "" {
		return KPI{}, errors.New("quarter is required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return KPI{}, errors.New("title is required")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE kpis
		SET quarter = ?, title = ?, description = ?, weight = ?, target_metric = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`, input.Quarter, input.Title, input.Description, input.Weight, input.TargetMetric, input.ID, input.UserID)
	if err != nil {
		return KPI{}, fmt.Errorf("update kpi: %w", err)
	}
	return s.Get(ctx, input.UserID, input.ID)
}

func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM kpis WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete kpi: %w", err)
	}
	return nil
}
