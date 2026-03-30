package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAuditList_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "event_type", "entity_type", "entity_id", "payload_json", "created_at"}).
		AddRow(1, "ConsentCreated", "consent", 10, `{"consent_id":10}`, "2026-01-01 00:00:00").
		AddRow(2, "ConsentRevoked", "consent", 10, `{"consent_id":10}`, "2026-01-02 00:00:00")
	mock.ExpectQuery("SELECT id, event_type").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/audit-events", nil)
	w := httptest.NewRecorder()
	AuditHandler{DB: db}.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result []AuditEvent
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result))
	}
}

func TestAuditList_Empty(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery("SELECT id, event_type").WillReturnRows(
		sqlmock.NewRows([]string{"id", "event_type", "entity_type", "entity_id", "payload_json", "created_at"}),
	)

	req := httptest.NewRequest(http.MethodGet, "/audit-events", nil)
	w := httptest.NewRecorder()
	AuditHandler{DB: db}.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuditList_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery("SELECT id, event_type").WillReturnError(errors.New("db down"))

	req := httptest.NewRequest(http.MethodGet, "/audit-events", nil)
	w := httptest.NewRecorder()
	AuditHandler{DB: db}.List(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestAuditList_RowsErr(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "event_type", "entity_type", "entity_id", "payload_json", "created_at"}).
		AddRow(1, "ConsentCreated", "consent", 1, `{}`, "2026-01-01").
		RowError(0, errors.New("row error"))
	mock.ExpectQuery("SELECT id, event_type").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/audit-events", nil)
	w := httptest.NewRecorder()
	AuditHandler{DB: db}.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestAuditList_ScanError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "event_type", "entity_type", "entity_id", "payload_json", "created_at"}).
		AddRow("not-a-number", "ConsentCreated", "consent", 1, `{}`, "2026-01-01")
	mock.ExpectQuery("SELECT id, event_type").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/audit-events", nil)
	w := httptest.NewRecorder()
	AuditHandler{DB: db}.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
