package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// ── Record ───────────────────────────────────────────────────────────────────

func TestLineageRecord_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectExec("INSERT INTO data_lineage").
		WithArgs(uint64(1), "consent_created", "mono-mvc", "db", "marketing", nil, "{}").
		WillReturnResult(sqlmock.NewResult(10, 1))

	body, _ := json.Marshal(createLineageReq{
		SubjectID:   1,
		Operation:   "consent_created",
		Source:      "mono-mvc",
		Destination: "db",
		Purpose:     "marketing",
		PayloadJSON: "{}",
	})
	req := httptest.NewRequest(http.MethodPost, "/lineage", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	LineageHandler{DB: db}.Record(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestLineageRecord_InvalidBody(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/lineage", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	LineageHandler{DB: db}.Record(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLineageRecord_MissingFields(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	body, _ := json.Marshal(map[string]any{"subject_id": 1})
	req := httptest.NewRequest(http.MethodPost, "/lineage", bytes.NewReader(body))
	w := httptest.NewRecorder()
	LineageHandler{DB: db}.Record(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

func TestLineageRecord_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectExec("INSERT INTO data_lineage").WillReturnError(errors.New("insert failed"))

	body, _ := json.Marshal(createLineageReq{
		SubjectID:   1,
		Operation:   "consent_created",
		Source:      "mono-mvc",
		Destination: "db",
		Purpose:     "marketing",
		PayloadJSON: "{}",
	})
	req := httptest.NewRequest(http.MethodPost, "/lineage", bytes.NewReader(body))
	w := httptest.NewRecorder()
	LineageHandler{DB: db}.Record(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Export ────────────────────────────────────────────────────────────────────

func TestLineageExport_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "subject_id", "operation", "source", "destination", "purpose", "consent_id", "payload_json", "created_at"}).
		AddRow(1, 1, "consent_created", "mono-mvc", "db", "marketing", int64(5), "{}", "2026-01-01 00:00:00").
		AddRow(2, 1, "data_exported", "mono-mvc", "s3", "analytics", nil, "{}", "2026-01-02 00:00:00")
	mock.ExpectQuery("SELECT id, subject_id").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/lineage/export/1", nil)
	req = chiCtx(req, "subject_id", "1")
	w := httptest.NewRecorder()

	LineageHandler{DB: db}.Export(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result []LineageEvent
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result))
	}
	if result[0].ConsentID == nil || *result[0].ConsentID != 5 {
		t.Fatal("expected consent_id=5 for first event")
	}
	if result[1].ConsentID != nil {
		t.Fatal("expected nil consent_id for second event")
	}
}

func TestLineageExport_Empty(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery("SELECT id, subject_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "subject_id", "operation", "source", "destination", "purpose", "consent_id", "payload_json", "created_at"}),
	)

	req := httptest.NewRequest(http.MethodGet, "/lineage/export/999", nil)
	req = chiCtx(req, "subject_id", "999")
	w := httptest.NewRecorder()

	LineageHandler{DB: db}.Export(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result []LineageEvent
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 events, got %d", len(result))
	}
}

func TestLineageExport_InvalidSubjectID(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/lineage/export/abc", nil)
	req = chiCtx(req, "subject_id", "abc")
	w := httptest.NewRecorder()

	LineageHandler{DB: db}.Export(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLineageExport_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery("SELECT id, subject_id").WillReturnError(errors.New("connection lost"))

	req := httptest.NewRequest(http.MethodGet, "/lineage/export/1", nil)
	req = chiCtx(req, "subject_id", "1")
	w := httptest.NewRecorder()

	LineageHandler{DB: db}.Export(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
