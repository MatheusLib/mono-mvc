package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type Consent struct {
	ID        uint64  `json:"id"`
	UserID    uint64  `json:"user_id"`
	PolicyID  uint64  `json:"policy_id"`
	Purpose   string  `json:"purpose"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
	RevokedAt *string `json:"revoked_at,omitempty"`
}

type createConsentReq struct {
	UserID   uint64 `json:"user_id"`
	PolicyID uint64 `json:"policy_id"`
	Purpose  string `json:"purpose"`
}

type ConsentsHandler struct {
	DB *sql.DB
}

func (h ConsentsHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, user_id, policy_id, purpose, status, created_at, revoked_at
		FROM consents
		ORDER BY id
	`)
	if err != nil {
		http.Error(w, "db unreachable", http.StatusServiceUnavailable)
		return
	}
	defer rows.Close()

	consents := make([]Consent, 0)
	for rows.Next() {
		var c Consent
		var revokedAt sql.NullString
		if err := rows.Scan(&c.ID, &c.UserID, &c.PolicyID, &c.Purpose, &c.Status, &c.CreatedAt, &revokedAt); err != nil {
			http.Error(w, "query error", http.StatusInternalServerError)
			return
		}
		if revokedAt.Valid {
			c.RevokedAt = &revokedAt.String
		}
		consents = append(consents, c)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(consents)
}

func (h ConsentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req createConsentReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.UserID == 0 || req.PolicyID == 0 || req.Purpose == "" {
		http.Error(w, "user_id, policy_id and purpose are required", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "db error", http.StatusServiceUnavailable)
		return
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO consents (user_id, policy_id, purpose, status)
		VALUES (?, ?, ?, 'active')
	`, req.UserID, req.PolicyID, req.Purpose)
	if err != nil {
		http.Error(w, "insert error", http.StatusInternalServerError)
		return
	}

	consentID, _ := res.LastInsertId()

	payload, _ := json.Marshal(map[string]any{
		"consent_id": consentID,
		"user_id":    req.UserID,
		"policy_id":  req.PolicyID,
		"purpose":    req.Purpose,
	})
	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_events (event_type, entity_type, entity_id, payload_json)
		VALUES ('ConsentCreated', 'consent', ?, ?)
	`, consentID, string(payload))
	if err != nil {
		http.Error(w, "audit error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "commit error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":         consentID,
		"user_id":    req.UserID,
		"policy_id":  req.PolicyID,
		"purpose":    req.Purpose,
		"status":     "active",
		"created_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h ConsentsHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	documentID, err := strconv.ParseUint(chi.URLParam(r, "document_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid document_id", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "db error", http.StatusServiceUnavailable)
		return
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		UPDATE consents
		SET status = 'revoked', revoked_at = NOW()
		WHERE id = ? AND status = 'active'
	`, documentID)
	if err != nil {
		http.Error(w, "update error", http.StatusInternalServerError)
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.Error(w, "consent not found or already revoked", http.StatusNotFound)
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"consent_id": documentID,
		"revoked_at": time.Now().UTC().Format(time.RFC3339),
	})
	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_events (event_type, entity_type, entity_id, payload_json)
		VALUES ('ConsentRevoked', 'consent', ?, ?)
	`, documentID, string(payload))
	if err != nil {
		http.Error(w, "audit error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "commit error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
