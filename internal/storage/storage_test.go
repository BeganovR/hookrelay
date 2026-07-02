//go:build integration

package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"hookrelay/internal/domain"
	"hookrelay/internal/storage"
)

func setupDB(t *testing.T) (*storage.DB, func()) {
	t.Helper()
	ctx := context.Background()

	pgC, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("hookrelay"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	connStr, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgC.Terminate(ctx)
		t.Fatalf("connection string: %v", err)
	}

	if err := storage.RunMigrations(connStr); err != nil {
		_ = pgC.Terminate(ctx)
		t.Fatalf("run migrations: %v", err)
	}

	pool, err := storage.Connect(connStr)
	if err != nil {
		_ = pgC.Terminate(ctx)
		t.Fatalf("connect pool: %v", err)
	}

	db := storage.New(pool)

	return db, func() {
		pool.Close()
		_ = pgC.Terminate(ctx)
	}
}

func TestProject_CRUD(t *testing.T) {
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	proj, err := db.CreateProject(ctx, "my-project", "integration test project")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if proj.ID == "" {
		t.Fatal("expected non-empty project ID")
	}
	if proj.Name != "my-project" {
		t.Errorf("name: want my-project, got %q", proj.Name)
	}

	got, err := db.GetProject(ctx, proj.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.ID != proj.ID {
		t.Errorf("GetProject ID mismatch: want %q, got %q", proj.ID, got.ID)
	}

	updated, err := db.UpdateProject(ctx, proj.ID, "renamed", "new description")
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	if updated.Name != "renamed" {
		t.Errorf("updated name: want renamed, got %q", updated.Name)
	}

	if err := db.DeleteProject(ctx, proj.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	_, err = db.GetProject(ctx, proj.ID)
	if err == nil {
		t.Error("expected ErrNotFound after delete, got nil")
	}
}

func TestAPIKey_CreateAndLookup(t *testing.T) {
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	proj, err := db.CreateProject(ctx, "key-test-proj", "")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	hash := "a" + string(make([]byte, 63))
	for i := range hash {
		if hash[i] == 0 {
			hash = hash[:i] + "b" + hash[i+1:]
		}
	}
	hash = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	key, err := db.CreateAPIKey(ctx, proj.ID, "default", hash, "hr_abcde")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if key.ProjectID != proj.ID {
		t.Errorf("key.ProjectID: want %q, got %q", proj.ID, key.ProjectID)
	}

	found, err := db.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		t.Fatalf("GetAPIKeyByHash: %v", err)
	}
	if found.ID != key.ID {
		t.Errorf("key ID mismatch: want %q, got %q", key.ID, found.ID)
	}

	keys, err := db.ListAPIKeys(ctx, proj.ID)
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("expected 1 key, got %d", len(keys))
	}

	if err := db.DeleteAPIKey(ctx, proj.ID, key.ID); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}
	_, err = db.GetAPIKeyByHash(ctx, hash)
	if err == nil {
		t.Error("expected ErrNotFound after delete")
	}
}

func TestSource_CRUD(t *testing.T) {
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	proj, _ := db.CreateProject(ctx, "src-proj", "")

	src, err := db.CreateSource(ctx, storage.CreateSourceParams{
		ProjectID:    proj.ID,
		Name:         "github-webhooks",
		UID:          "src-gh-001",
		VerifierType: domain.VerifierNoop,
	})
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}
	if src.UID != "src-gh-001" {
		t.Errorf("UID: want src-gh-001, got %q", src.UID)
	}
	if !src.IsActive {
		t.Error("expected source to be active by default")
	}

	byUID, err := db.GetSourceByUID(ctx, "src-gh-001")
	if err != nil {
		t.Fatalf("GetSourceByUID: %v", err)
	}
	if byUID.ID != src.ID {
		t.Errorf("ID mismatch: want %q, got %q", src.ID, byUID.ID)
	}

	active := false
	updated, err := db.UpdateSource(ctx, proj.ID, src.ID, storage.UpdateSourceParams{IsActive: &active})
	if err != nil {
		t.Fatalf("UpdateSource: %v", err)
	}
	if updated.IsActive {
		t.Error("expected source to be inactive after update")
	}

	srcs, err := db.ListSources(ctx, proj.ID)
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if len(srcs) != 1 {
		t.Errorf("expected 1 source, got %d", len(srcs))
	}
}

func TestEvent_Idempotency(t *testing.T) {
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	proj, _ := db.CreateProject(ctx, "event-proj", "")
	src, _ := db.CreateSource(ctx, storage.CreateSourceParams{
		ProjectID:    proj.ID,
		Name:         "src",
		UID:          "src-evt-001",
		VerifierType: domain.VerifierNoop,
	})

	idemKey := "idem-key-abc-123"
	params := storage.CreateEventParams{
		ProjectID:      proj.ID,
		SourceID:       &src.ID,
		EventType:      "order.created",
		Payload:        []byte(`{"order_id":42}`),
		SenderIP:       "127.0.0.1",
		IdempotencyKey: &idemKey,
	}

	e1, err := db.CreateEvent(ctx, params)
	if err != nil {
		t.Fatalf("first CreateEvent: %v", err)
	}

	e2, err := db.CreateEvent(ctx, params)
	if err != nil {
		t.Fatalf("second CreateEvent (idempotent): %v", err)
	}

	if e1.ID != e2.ID {
		t.Errorf("idempotent inserts should return same event: %q vs %q", e1.ID, e2.ID)
	}
	if !e1.IsNew {
		t.Error("first insert should report IsNew=true")
	}
	if e2.IsNew {
		t.Error("duplicate insert should report IsNew=false")
	}
}

func TestEvent_WithoutIdempotencyKey(t *testing.T) {
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	proj, _ := db.CreateProject(ctx, "event-proj2", "")
	src, _ := db.CreateSource(ctx, storage.CreateSourceParams{
		ProjectID:    proj.ID,
		Name:         "src",
		UID:          "src-evt-002",
		VerifierType: domain.VerifierNoop,
	})

	params := storage.CreateEventParams{
		ProjectID: proj.ID,
		SourceID:  &src.ID,
		EventType: "ping",
		Payload:   []byte(`{}`),
		SenderIP:  "127.0.0.1",
	}

	e1, err := db.CreateEvent(ctx, params)
	if err != nil {
		t.Fatalf("first CreateEvent: %v", err)
	}
	e2, err := db.CreateEvent(ctx, params)
	if err != nil {
		t.Fatalf("second CreateEvent: %v", err)
	}
	if e1.ID == e2.ID {
		t.Error("events without idempotency key should be distinct")
	}
	if !e1.IsNew || !e2.IsNew {
		t.Error("events without idempotency key should always report IsNew=true")
	}
}

func TestDelivery_ClaimAndComplete(t *testing.T) {
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	proj, _ := db.CreateProject(ctx, "delivery-proj", "")
	src, _ := db.CreateSource(ctx, storage.CreateSourceParams{
		ProjectID:    proj.ID,
		Name:         "src",
		UID:          "src-del-001",
		VerifierType: domain.VerifierNoop,
	})
	ep, err := db.CreateEndpoint(ctx, storage.CreateEndpointParams{
		ProjectID: proj.ID,
		Name:      "my-endpoint",
		URL:       "https://example.com/webhook",
	})
	if err != nil {
		t.Fatalf("CreateEndpoint: %v", err)
	}

	event, _ := db.CreateEvent(ctx, storage.CreateEventParams{
		ProjectID: proj.ID,
		SourceID:  &src.ID,
		EventType: "test.event",
		Payload:   []byte(`{"x":1}`),
		SenderIP:  "127.0.0.1",
	})

	delivery, err := db.CreateDelivery(ctx, storage.CreateDeliveryParams{
		ProjectID:  proj.ID,
		EventID:    event.ID,
		EndpointID: ep.ID,
		MaxRetries: 3,
		RetryType:  domain.RetryExponential,
	})
	if err != nil {
		t.Fatalf("CreateDelivery: %v", err)
	}
	if delivery.Status != "pending" {
		t.Errorf("expected pending status, got %q", delivery.Status)
	}

	claimed, err := db.ClaimPendingDeliveries(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimPendingDeliveries: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("expected 1 claimed delivery, got %d", len(claimed))
	}
	if claimed[0].EventDelivery.ID != delivery.ID {
		t.Errorf("claimed wrong delivery: %q vs %q", claimed[0].EventDelivery.ID, delivery.ID)
	}

	if err := db.MarkDeliverySuccess(ctx, delivery.ID); err != nil {
		t.Fatalf("MarkDeliverySuccess: %v", err)
	}

	got, err := db.GetDelivery(ctx, proj.ID, delivery.ID)
	if err != nil {
		t.Fatalf("GetDelivery: %v", err)
	}
	if got.Status != "success" {
		t.Errorf("expected success status, got %q", got.Status)
	}
}

func TestDelivery_ClaimBumpsScheduledAt(t *testing.T) {
	db, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	proj, _ := db.CreateProject(ctx, "claim-proj", "")
	src, _ := db.CreateSource(ctx, storage.CreateSourceParams{
		ProjectID:    proj.ID,
		Name:         "src",
		UID:          "src-claim-001",
		VerifierType: domain.VerifierNoop,
	})
	ep, _ := db.CreateEndpoint(ctx, storage.CreateEndpointParams{
		ProjectID: proj.ID,
		Name:      "ep",
		URL:       "https://example.com/webhook",
	})
	event, _ := db.CreateEvent(ctx, storage.CreateEventParams{
		ProjectID: proj.ID,
		SourceID:  &src.ID,
		EventType: "test.event",
		Payload:   []byte(`{}`),
		SenderIP:  "127.0.0.1",
	})
	delivery, err := db.CreateDelivery(ctx, storage.CreateDeliveryParams{
		ProjectID:  proj.ID,
		EventID:    event.ID,
		EndpointID: ep.ID,
		MaxRetries: 3,
		RetryType:  domain.RetryExponential,
	})
	if err != nil {
		t.Fatalf("CreateDelivery: %v", err)
	}

	backdated := time.Now().Add(-1 * time.Hour)
	if _, err := db.Pool.Exec(ctx, `UPDATE deliveries SET next_retry_at = $1 WHERE id = $2`, backdated, delivery.ID); err != nil {
		t.Fatalf("backdate next_retry_at: %v", err)
	}

	claimed, err := db.ClaimPendingDeliveries(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimPendingDeliveries: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("expected 1 claimed delivery, got %d", len(claimed))
	}

	if claimed[0].ScheduledAt.Before(time.Now().Add(-1 * time.Minute)) {
		t.Errorf("expected scheduled_at to be bumped to claim time, got %v (was backdated to %v)", claimed[0].ScheduledAt, backdated)
	}

	stuckBefore := time.Now().Add(-30 * time.Second)
	n, err := db.RecoverStuckDeliveries(ctx, stuckBefore)
	if err != nil {
		t.Fatalf("RecoverStuckDeliveries: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 deliveries recovered (still genuinely processing), got %d", n)
	}
}
