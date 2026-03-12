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
	ID             int64     `json:"id"`
	UserID         int64     `json:"userId"`
	Quarter        string    `json:"quarter"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Weight         int       `json:"weight"`
	TargetMetric   string    `json:"targetMetric"`
	ParentKPIID    *int64    `json:"parentKpiId"`
	ProgressQ1     int       `json:"progressQ1"`
	ProgressQ2     int       `json:"progressQ2"`
	ProgressQ3     int       `json:"progressQ3"`
	ProgressQ4     int       `json:"progressQ4"`
	AnnualProgress int       `json:"annualProgress"`
	Children       []KPI     `json:"children,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Service struct {
	db *sql.DB
}

type ProgressUpdate struct {
	ProgressQ1 *int `json:"progressQ1"`
	ProgressQ2 *int `json:"progressQ2"`
	ProgressQ3 *int `json:"progressQ3"`
	ProgressQ4 *int `json:"progressQ4"`
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, userID int64, quarter string) ([]KPI, error) {
	query := `
		SELECT id, user_id, quarter, title, description, weight, target_metric,
		       parent_kpi_id, progress_q1, progress_q2, progress_q3, progress_q4, annual_progress,
		       created_at, updated_at
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
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Quarter, &item.Title, &item.Description,
			&item.Weight, &item.TargetMetric, &item.ParentKPIID,
			&item.ProgressQ1, &item.ProgressQ2, &item.ProgressQ3, &item.ProgressQ4, &item.AnnualProgress,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		kpis = append(kpis, item)
	}
	return kpis, rows.Err()
}

// ListWithHierarchy returns KPIs organized as a tree structure
func (s *Service) ListWithHierarchy(ctx context.Context, userID int64, quarter string) ([]KPI, error) {
	kpis, err := s.List(ctx, userID, quarter)
	if err != nil {
		return nil, err
	}
	return buildHierarchy(kpis), nil
}

// buildHierarchy organizes flat KPI list into tree structure
func buildHierarchy(kpis []KPI) []KPI {
	nodes := make(map[int64]KPI, len(kpis))
	childrenByParent := make(map[int64][]int64)
	rootIDs := make([]int64, 0)

	for _, item := range kpis {
		item.Children = nil
		nodes[item.ID] = item
	}

	for _, item := range kpis {
		if item.ParentKPIID == nil {
			rootIDs = append(rootIDs, item.ID)
			continue
		}

		if _, ok := nodes[*item.ParentKPIID]; !ok {
			rootIDs = append(rootIDs, item.ID)
			continue
		}
		childrenByParent[*item.ParentKPIID] = append(childrenByParent[*item.ParentKPIID], item.ID)
	}

	var buildNode func(id int64) KPI
	buildNode = func(id int64) KPI {
		node := nodes[id]
		childIDs := childrenByParent[id]
		if len(childIDs) == 0 {
			node.Children = nil
			return node
		}

		node.Children = make([]KPI, 0, len(childIDs))
		for _, childID := range childIDs {
			node.Children = append(node.Children, buildNode(childID))
		}
		return node
	}

	roots := make([]KPI, 0, len(rootIDs))
	for _, rootID := range rootIDs {
		roots = append(roots, buildNode(rootID))
	}

	return roots
}

// GetChildren returns all direct children of a parent KPI
func (s *Service) GetChildren(ctx context.Context, userID int64, parentID int64) ([]KPI, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, quarter, title, description, weight, target_metric,
		       parent_kpi_id, progress_q1, progress_q2, progress_q3, progress_q4, annual_progress,
		       created_at, updated_at
		FROM kpis
		WHERE user_id = ? AND parent_kpi_id = ?
		ORDER BY created_at DESC
	`, userID, parentID)
	if err != nil {
		return nil, fmt.Errorf("get children: %w", err)
	}
	defer rows.Close()

	var kpis []KPI
	for rows.Next() {
		var item KPI
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Quarter, &item.Title, &item.Description,
			&item.Weight, &item.TargetMetric, &item.ParentKPIID,
			&item.ProgressQ1, &item.ProgressQ2, &item.ProgressQ3, &item.ProgressQ4, &item.AnnualProgress,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		kpis = append(kpis, item)
	}
	return kpis, rows.Err()
}

func (s *Service) Create(ctx context.Context, input KPI) (KPI, error) {
	if strings.TrimSpace(input.Title) == "" {
		return KPI{}, errors.New("title is required")
	}

	for _, value := range []int{input.ProgressQ1, input.ProgressQ2, input.ProgressQ3, input.ProgressQ4} {
		if err := validateProgress(value); err != nil {
			return KPI{}, err
		}
	}

	// Validate parent ownership if specified and inherit parent quarter.
	if input.ParentKPIID != nil {
		parent, err := s.Get(ctx, input.UserID, *input.ParentKPIID)
		if err != nil {
			return KPI{}, errors.New("parent KPI not found or access denied")
		}
		input.Quarter = parent.Quarter
	}
	if strings.TrimSpace(input.Quarter) == "" {
		return KPI{}, errors.New("quarter is required")
	}
	input.AnnualProgress = calculateAnnualProgress(input.ProgressQ1, input.ProgressQ2, input.ProgressQ3, input.ProgressQ4)

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO kpis (user_id, quarter, title, description, weight, target_metric, parent_kpi_id,
		                  progress_q1, progress_q2, progress_q3, progress_q4, annual_progress)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, input.UserID, input.Quarter, input.Title, input.Description, input.Weight, input.TargetMetric,
		input.ParentKPIID, input.ProgressQ1, input.ProgressQ2, input.ProgressQ3, input.ProgressQ4, input.AnnualProgress)
	if err != nil {
		return KPI{}, fmt.Errorf("create kpi: %w", err)
	}

	id, _ := result.LastInsertId()
	return s.Get(ctx, input.UserID, id)
}

// CreateSubKPI creates a KPI under a parent, inheriting the quarter
func (s *Service) CreateSubKPI(ctx context.Context, userID int64, parentID int64, input KPI) (KPI, error) {
	input.UserID = userID
	input.ParentKPIID = &parentID
	return s.Create(ctx, input)
}

func (s *Service) Get(ctx context.Context, userID, id int64) (KPI, error) {
	var item KPI
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, quarter, title, description, weight, target_metric,
		       parent_kpi_id, progress_q1, progress_q2, progress_q3, progress_q4, annual_progress,
		       created_at, updated_at
		FROM kpis
		WHERE user_id = ? AND id = ?
	`, userID, id).Scan(
		&item.ID, &item.UserID, &item.Quarter, &item.Title, &item.Description,
		&item.Weight, &item.TargetMetric, &item.ParentKPIID,
		&item.ProgressQ1, &item.ProgressQ2, &item.ProgressQ3, &item.ProgressQ4, &item.AnnualProgress,
		&item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *Service) Update(ctx context.Context, input KPI) (KPI, error) {
	if strings.TrimSpace(input.Quarter) == "" {
		return KPI{}, errors.New("quarter is required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return KPI{}, errors.New("title is required")
	}
	for _, value := range []int{input.ProgressQ1, input.ProgressQ2, input.ProgressQ3, input.ProgressQ4} {
		if err := validateProgress(value); err != nil {
			return KPI{}, err
		}
	}

	existing, err := s.Get(ctx, input.UserID, input.ID)
	if err != nil {
		return KPI{}, err
	}

	if input.ParentKPIID != nil {
		if *input.ParentKPIID == input.ID {
			return KPI{}, errors.New("a KPI cannot be parent of itself")
		}

		parent, err := s.Get(ctx, input.UserID, *input.ParentKPIID)
		if err != nil {
			return KPI{}, errors.New("parent KPI not found or access denied")
		}
		if err := s.ensureNoCycle(ctx, input.UserID, input.ID, *input.ParentKPIID); err != nil {
			return KPI{}, err
		}

		// Child quarter is tied to parent quarter for consistency.
		input.Quarter = parent.Quarter
	}

	// Calculate annual progress from quarterly values
	input.AnnualProgress = calculateAnnualProgress(input.ProgressQ1, input.ProgressQ2, input.ProgressQ3, input.ProgressQ4)

	_, err = s.db.ExecContext(ctx, `
		UPDATE kpis
		SET quarter = ?, title = ?, description = ?, weight = ?, target_metric = ?,
		    parent_kpi_id = ?, progress_q1 = ?, progress_q2 = ?, progress_q3 = ?, progress_q4 = ?,
		    annual_progress = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`, input.Quarter, input.Title, input.Description, input.Weight, input.TargetMetric,
		input.ParentKPIID, input.ProgressQ1, input.ProgressQ2, input.ProgressQ3, input.ProgressQ4,
		input.AnnualProgress, input.ID, input.UserID)
	if err != nil {
		return KPI{}, fmt.Errorf("update kpi: %w", err)
	}

	// Recalculate parent progress if this KPI has a parent
	kpi, err := s.Get(ctx, input.UserID, input.ID)
	if err != nil {
		return KPI{}, err
	}

	oldParent := existing.ParentKPIID
	newParent := kpi.ParentKPIID
	if oldParent != nil && (newParent == nil || *newParent != *oldParent) {
		if err := s.RecalculateParentProgress(ctx, input.UserID, *oldParent); err != nil {
			return KPI{}, err
		}
	}
	if newParent != nil {
		if err := s.RecalculateParentProgress(ctx, input.UserID, *newParent); err != nil {
			return KPI{}, err
		}
	}

	return kpi, nil
}

// UpdateProgress updates one or more quarterly values and recalculates annual/parent progress.
func (s *Service) UpdateProgress(ctx context.Context, userID int64, id int64, update ProgressUpdate) (KPI, error) {
	if update.ProgressQ1 == nil && update.ProgressQ2 == nil && update.ProgressQ3 == nil && update.ProgressQ4 == nil {
		return KPI{}, errors.New("at least one quarterly progress field is required")
	}
	for _, value := range []*int{update.ProgressQ1, update.ProgressQ2, update.ProgressQ3, update.ProgressQ4} {
		if value == nil {
			continue
		}
		if err := validateProgress(*value); err != nil {
			return KPI{}, err
		}
	}

	item, err := s.Get(ctx, userID, id)
	if err != nil {
		return KPI{}, err
	}
	if update.ProgressQ1 != nil {
		item.ProgressQ1 = *update.ProgressQ1
	}
	if update.ProgressQ2 != nil {
		item.ProgressQ2 = *update.ProgressQ2
	}
	if update.ProgressQ3 != nil {
		item.ProgressQ3 = *update.ProgressQ3
	}
	if update.ProgressQ4 != nil {
		item.ProgressQ4 = *update.ProgressQ4
	}
	item.AnnualProgress = calculateAnnualProgress(item.ProgressQ1, item.ProgressQ2, item.ProgressQ3, item.ProgressQ4)

	_, err = s.db.ExecContext(ctx, `
		UPDATE kpis
		SET progress_q1 = ?, progress_q2 = ?, progress_q3 = ?, progress_q4 = ?, annual_progress = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`, item.ProgressQ1, item.ProgressQ2, item.ProgressQ3, item.ProgressQ4, item.AnnualProgress, id, userID)
	if err != nil {
		return KPI{}, fmt.Errorf("update progress: %w", err)
	}

	// Recalculate ancestor progress if this KPI has a parent.
	if item.ParentKPIID != nil {
		if err := s.RecalculateParentProgress(ctx, userID, *item.ParentKPIID); err != nil {
			return KPI{}, err
		}
	}

	return s.Get(ctx, userID, id)
}

// RecalculateParentProgress aggregates child progress to parent using simple average
func (s *Service) RecalculateParentProgress(ctx context.Context, userID int64, parentID int64) error {
	children, err := s.GetChildren(ctx, userID, parentID)
	if err != nil {
		return err
	}

	if len(children) == 0 {
		return nil
	}

	// Calculate simple average of children's progress for each quarter
	var totalQ1, totalQ2, totalQ3, totalQ4 int
	for _, child := range children {
		totalQ1 += child.ProgressQ1
		totalQ2 += child.ProgressQ2
		totalQ3 += child.ProgressQ3
		totalQ4 += child.ProgressQ4
	}

	count := len(children)
	avgQ1 := totalQ1 / count
	avgQ2 := totalQ2 / count
	avgQ3 := totalQ3 / count
	avgQ4 := totalQ4 / count
	annualProgress := calculateAnnualProgress(avgQ1, avgQ2, avgQ3, avgQ4)

	_, err = s.db.ExecContext(ctx, `
		UPDATE kpis
		SET progress_q1 = ?, progress_q2 = ?, progress_q3 = ?, progress_q4 = ?,
		    annual_progress = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`, avgQ1, avgQ2, avgQ3, avgQ4, annualProgress, parentID, userID)
	if err != nil {
		return fmt.Errorf("recalculate parent progress: %w", err)
	}

	// Check if parent has a grandparent and recurse
	parent, err := s.Get(ctx, userID, parentID)
	if err != nil {
		return err
	}
	if parent.ParentKPIID != nil {
		return s.RecalculateParentProgress(ctx, userID, *parent.ParentKPIID)
	}

	return nil
}

// calculateAnnualProgress computes simple average of quarterly progress
func calculateAnnualProgress(q1, q2, q3, q4 int) int {
	return (q1 + q2 + q3 + q4) / 4
}

func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	// Get KPI to check if it has a parent (for recalculation after delete)
	kpi, err := s.Get(ctx, userID, id)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	parentID := kpi.ParentKPIID

	_, err = s.db.ExecContext(ctx, `DELETE FROM kpis WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete kpi: %w", err)
	}

	// Recalculate parent progress after deletion
	if parentID != nil {
		if err := s.RecalculateParentProgress(ctx, userID, *parentID); err != nil {
			// Log error but don't fail the delete
			return nil
		}
	}

	return nil
}

func (s *Service) ensureNoCycle(ctx context.Context, userID, kpiID, parentID int64) error {
	currentID := parentID
	seen := map[int64]struct{}{kpiID: {}}

	for {
		if _, exists := seen[currentID]; exists {
			return errors.New("invalid parent relationship: cycle detected")
		}
		seen[currentID] = struct{}{}

		parent, err := s.Get(ctx, userID, currentID)
		if err != nil {
			return err
		}
		if parent.ParentKPIID == nil {
			return nil
		}
		currentID = *parent.ParentKPIID
	}
}

func validateProgress(progress int) error {
	if progress < 0 || progress > 100 {
		return errors.New("progress must be between 0 and 100")
	}
	return nil
}
