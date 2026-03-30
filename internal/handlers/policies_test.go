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

// ── List ──────────────────────────────────────────────────────────────────────

func TestPoliciesList_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "version", "content_hash", "created_at"}).
		AddRow(1, "v1.0", "abc123", "2026-01-01 00:00:00").
		AddRow(2, "v2.0", "def456", "2026-02-01 00:00:00")
	mock.ExpectQuery("SELECT id, version").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/policies", nil)
	w := httptest.NewRecorder()
	PoliciesHandler{DB: db}.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result []Policy
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(result))
	}
}

func TestPoliciesList_Empty(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery("SELECT id, version").WillReturnRows(
		sqlmock.NewRows([]string{"id", "version", "content_hash", "created_at"}),
	)

	req := httptest.NewRequest(http.MethodGet, "/policies", nil)
	w := httptest.NewRecorder()
	PoliciesHandler{DB: db}.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestPoliciesList_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery("SELECT id, version").WillReturnError(errors.New("db down"))

	req := httptest.NewRequest(http.MethodGet, "/policies", nil)
	w := httptest.NewRecorder()
	PoliciesHandler{DB: db}.List(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestPoliciesList_ScanError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "version", "content_hash", "created_at"}).
		AddRow("not-a-number", "v1.0", "abc123", "2026-01-01")
	mock.ExpectQuery("SELECT id, version").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/policies", nil)
	w := httptest.NewRecorder()
	PoliciesHandler{DB: db}.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestPoliciesList_RowsErr(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "version", "content_hash", "created_at"}).
		AddRow(1, "v1.0", "abc", "2026-01-01").
		RowError(0, errors.New("row error"))
	mock.ExpectQuery("SELECT id, version").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/policies", nil)
	w := httptest.NewRecorder()
	PoliciesHandler{DB: db}.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestPoliciesCreate_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectExec("INSERT INTO policies").
		WithArgs("v3.0", "hash999").
		WillReturnResult(sqlmock.NewResult(3, 1))

	body, _ := json.Marshal(createPolicyReq{Version: "v3.0", ContentHash: "hash999"})
	req := httptest.NewRequest(http.MethodPost, "/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	PoliciesHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestPoliciesCreate_InvalidBody(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/policies", bytes.NewReader([]byte("bad json")))
	w := httptest.NewRecorder()
	PoliciesHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPoliciesCreate_MissingFields(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	body, _ := json.Marshal(map[string]any{"version": "v1.0"})
	req := httptest.NewRequest(http.MethodPost, "/policies", bytes.NewReader(body))
	w := httptest.NewRecorder()
	PoliciesHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPoliciesCreate_InsertError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectExec("INSERT INTO policies").WillReturnError(errors.New("duplicate version"))

	body, _ := json.Marshal(createPolicyReq{Version: "v1.0", ContentHash: "abc"})
	req := httptest.NewRequest(http.MethodPost, "/policies", bytes.NewReader(body))
	w := httptest.NewRecorder()
	PoliciesHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
