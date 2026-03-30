package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
)

func chiCtx(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestConsentsList_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "user_id", "policy_id", "purpose", "status", "created_at", "revoked_at"}).
		AddRow(1, 10, 1, "marketing", "active", "2026-01-01 00:00:00", nil).
		AddRow(2, 11, 1, "analytics", "revoked", "2026-01-02 00:00:00", "2026-01-03 00:00:00")
	mock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)

	h := ConsentsHandler{DB: db}
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result []Consent
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 consents, got %d", len(result))
	}
	if result[1].RevokedAt == nil {
		t.Fatal("expected revoked_at to be set for second consent")
	}
}

func TestConsentsList_Empty(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery("SELECT id, user_id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "user_id", "policy_id", "purpose", "status", "created_at", "revoked_at"}),
	)

	h := ConsentsHandler{DB: db}
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestConsentsList_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery("SELECT id, user_id").WillReturnError(errors.New("connection lost"))

	h := ConsentsHandler{DB: db}
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestConsentsList_ScanError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "user_id", "policy_id", "purpose", "status", "created_at", "revoked_at"}).
		AddRow("not-a-number", 10, 1, "marketing", "active", "2026-01-01", nil)
	mock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)

	h := ConsentsHandler{DB: db}
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestConsentsList_RowsErr(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "user_id", "policy_id", "purpose", "status", "created_at", "revoked_at"}).
		AddRow(1, 10, 1, "marketing", "active", "2026-01-01", nil).
		RowError(0, errors.New("row iteration error"))
	mock.ExpectQuery("SELECT id, user_id").WillReturnRows(rows)

	h := ConsentsHandler{DB: db}
	req := httptest.NewRequest(http.MethodGet, "/consents", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestConsentsCreate_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO consents").
		WithArgs(uint64(10), uint64(1), "marketing").
		WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec("INSERT INTO audit_events").
		WithArgs(int64(42), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	body, _ := json.Marshal(createConsentReq{UserID: 10, PolicyID: 1, Purpose: "marketing"})
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ConsentsHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestConsentsCreate_InvalidBody(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	ConsentsHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestConsentsCreate_MissingFields(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	body, _ := json.Marshal(map[string]any{"user_id": 1})
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewReader(body))
	w := httptest.NewRecorder()
	ConsentsHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestConsentsCreate_BeginTxError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	body, _ := json.Marshal(createConsentReq{UserID: 1, PolicyID: 1, Purpose: "x"})
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewReader(body))
	w := httptest.NewRecorder()
	ConsentsHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestConsentsCreate_InsertError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO consents").WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	body, _ := json.Marshal(createConsentReq{UserID: 1, PolicyID: 1, Purpose: "x"})
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewReader(body))
	w := httptest.NewRecorder()
	ConsentsHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestConsentsCreate_AuditError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO consents").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO audit_events").WillReturnError(errors.New("audit failed"))
	mock.ExpectRollback()

	body, _ := json.Marshal(createConsentReq{UserID: 1, PolicyID: 1, Purpose: "x"})
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewReader(body))
	w := httptest.NewRecorder()
	ConsentsHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestConsentsCreate_CommitError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO consents").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO audit_events").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	body, _ := json.Marshal(createConsentReq{UserID: 1, PolicyID: 1, Purpose: "x"})
	req := httptest.NewRequest(http.MethodPost, "/consents", bytes.NewReader(body))
	w := httptest.NewRecorder()
	ConsentsHandler{DB: db}.Create(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Revoke ────────────────────────────────────────────────────────────────────

func TestConsentsRevoke_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE consents").
		WithArgs(uint64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO audit_events").
		WithArgs(uint64(5), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPatch, "/consents/5/revoke", nil)
	req = chiCtx(req, "document_id", "5")
	w := httptest.NewRecorder()

	ConsentsHandler{DB: db}.Revoke(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestConsentsRevoke_InvalidDocumentID(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	req := httptest.NewRequest(http.MethodPatch, "/consents/abc/revoke", nil)
	req = chiCtx(req, "document_id", "abc")
	w := httptest.NewRecorder()

	ConsentsHandler{DB: db}.Revoke(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestConsentsRevoke_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE consents").
		WithArgs(uint64(99)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPatch, "/consents/99/revoke", nil)
	req = chiCtx(req, "document_id", "99")
	w := httptest.NewRecorder()

	ConsentsHandler{DB: db}.Revoke(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestConsentsRevoke_BeginTxError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	req := httptest.NewRequest(http.MethodPatch, "/consents/1/revoke", nil)
	req = chiCtx(req, "document_id", "1")
	w := httptest.NewRecorder()

	ConsentsHandler{DB: db}.Revoke(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestConsentsRevoke_UpdateError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE consents").WillReturnError(errors.New("update failed"))
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPatch, "/consents/1/revoke", nil)
	req = chiCtx(req, "document_id", "1")
	w := httptest.NewRecorder()

	ConsentsHandler{DB: db}.Revoke(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestConsentsRevoke_CommitError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE consents").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO audit_events").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	req := httptest.NewRequest(http.MethodPatch, "/consents/1/revoke", nil)
	req = chiCtx(req, "document_id", "1")
	w := httptest.NewRecorder()
	ConsentsHandler{DB: db}.Revoke(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestConsentsRevoke_AuditError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE consents").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO audit_events").WillReturnError(errors.New("audit failed"))
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPatch, "/consents/1/revoke", nil)
	req = chiCtx(req, "document_id", "1")
	w := httptest.NewRecorder()

	ConsentsHandler{DB: db}.Revoke(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
