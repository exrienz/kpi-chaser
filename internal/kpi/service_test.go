package kpi

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/example/kpi-chaser/internal/storage"
)

func setupService(t *testing.T) (*Service, int64) {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "kpi-test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := storage.Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	result, err := db.Exec(`INSERT INTO users (email, password_hash) VALUES (?, ?)`, "test@example.com", "hash")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("user id: %v", err)
	}

	return NewService(db), userID
}

func TestListWithHierarchySupportsNestedChildren(t *testing.T) {
	ctx := context.Background()
	svc, userID := setupService(t)

	root, err := svc.Create(ctx, KPI{UserID: userID, Quarter: "2026-Q1", Title: "Root"})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	child, err := svc.CreateSubKPI(ctx, userID, root.ID, KPI{Title: "Child"})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	if _, err := svc.CreateSubKPI(ctx, userID, child.ID, KPI{Title: "Grandchild"}); err != nil {
		t.Fatalf("create grandchild: %v", err)
	}

	hierarchy, err := svc.ListWithHierarchy(ctx, userID, "2026-Q1")
	if err != nil {
		t.Fatalf("list hierarchy: %v", err)
	}

	if len(hierarchy) != 1 {
		t.Fatalf("expected 1 root, got %d", len(hierarchy))
	}
	if len(hierarchy[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(hierarchy[0].Children))
	}
	if len(hierarchy[0].Children[0].Children) != 1 {
		t.Fatalf("expected 1 grandchild, got %d", len(hierarchy[0].Children[0].Children))
	}
}

func TestUpdateProgressRecalculatesParentAverages(t *testing.T) {
	ctx := context.Background()
	svc, userID := setupService(t)

	parent, err := svc.Create(ctx, KPI{UserID: userID, Quarter: "2026-Q1", Title: "Parent"})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	childA, err := svc.CreateSubKPI(ctx, userID, parent.ID, KPI{Title: "Child A"})
	if err != nil {
		t.Fatalf("create child A: %v", err)
	}
	childB, err := svc.CreateSubKPI(ctx, userID, parent.ID, KPI{Title: "Child B"})
	if err != nil {
		t.Fatalf("create child B: %v", err)
	}

	if _, err := svc.UpdateProgress(ctx, userID, childA.ID, ProgressUpdate{
		ProgressQ1: intPtr(100),
		ProgressQ2: intPtr(0),
		ProgressQ3: intPtr(0),
		ProgressQ4: intPtr(0),
	}); err != nil {
		t.Fatalf("update child A progress: %v", err)
	}

	if _, err := svc.UpdateProgress(ctx, userID, childB.ID, ProgressUpdate{
		ProgressQ1: intPtr(0),
		ProgressQ2: intPtr(100),
		ProgressQ3: intPtr(100),
		ProgressQ4: intPtr(100),
	}); err != nil {
		t.Fatalf("update child B progress: %v", err)
	}

	updatedParent, err := svc.Get(ctx, userID, parent.ID)
	if err != nil {
		t.Fatalf("get parent: %v", err)
	}

	if updatedParent.ProgressQ1 != 50 || updatedParent.ProgressQ2 != 50 || updatedParent.ProgressQ3 != 50 || updatedParent.ProgressQ4 != 50 {
		t.Fatalf("unexpected parent quarterly progress: %+v", updatedParent)
	}
	if updatedParent.AnnualProgress != 50 {
		t.Fatalf("expected parent annual progress 50, got %d", updatedParent.AnnualProgress)
	}
}

func TestUpdateRejectsParentCycle(t *testing.T) {
	ctx := context.Background()
	svc, userID := setupService(t)

	root, err := svc.Create(ctx, KPI{UserID: userID, Quarter: "2026-Q1", Title: "Root"})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	child, err := svc.CreateSubKPI(ctx, userID, root.ID, KPI{Title: "Child"})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	root.ParentKPIID = &child.ID
	if _, err := svc.Update(ctx, root); err == nil {
		t.Fatal("expected cycle validation error, got nil")
	}
}

func intPtr(v int) *int {
	return &v
}
