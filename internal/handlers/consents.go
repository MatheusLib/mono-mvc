package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Consent struct {
	ID       uint64 `json:"id"`
	UserID   uint64 `json:"user_id"`
	PolicyID uint64 `json:"policy_id"`
	Purpose  string `json:"purpose"`
	Status   string `json:"status"`
}

type ConsentsHandler struct {
	DB *sql.DB
}

func (h ConsentsHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, user_id, policy_id, purpose, status
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
		if err := rows.Scan(&c.ID, &c.UserID, &c.PolicyID, &c.Purpose, &c.Status); err != nil {
			http.Error(w, "query error", http.StatusInternalServerError)
			return
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
