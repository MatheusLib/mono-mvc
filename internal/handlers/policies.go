package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Policy struct {
	ID          uint64 `json:"id"`
	Version     string `json:"version"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
}

type createPolicyReq struct {
	Version     string `json:"version"`
	ContentHash string `json:"content_hash"`
}

type PoliciesHandler struct {
	DB *sql.DB
}

func (h PoliciesHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, version, content_hash, created_at
		FROM policies
		ORDER BY id
	`)
	if err != nil {
		http.Error(w, "db unreachable", http.StatusServiceUnavailable)
		return
	}
	defer rows.Close()

	policies := make([]Policy, 0)
	for rows.Next() {
		var p Policy
		if err := rows.Scan(&p.ID, &p.Version, &p.ContentHash, &p.CreatedAt); err != nil {
			http.Error(w, "query error", http.StatusInternalServerError)
			return
		}
		policies = append(policies, p)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(policies)
}

func (h PoliciesHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req createPolicyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Version == "" || req.ContentHash == "" {
		http.Error(w, "version and content_hash are required", http.StatusBadRequest)
		return
	}

	res, err := h.DB.ExecContext(ctx, `
		INSERT INTO policies (version, content_hash)
		VALUES (?, ?)
	`, req.Version, req.ContentHash)
	if err != nil {
		http.Error(w, "insert error", http.StatusInternalServerError)
		return
	}

	policyID, _ := res.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":           policyID,
		"version":      req.Version,
		"content_hash": req.ContentHash,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
	})
}
