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

type LineageEvent struct {
	ID          uint64  `json:"id"`
	SubjectID   uint64  `json:"subject_id"`
	Operation   string  `json:"operation"`
	Source      string  `json:"source"`
	Destination string  `json:"destination"`
	Purpose     string  `json:"purpose"`
	ConsentID   *uint64 `json:"consent_id,omitempty"`
	PayloadJSON string  `json:"payload_json"`
	CreatedAt   string  `json:"created_at"`
}

type createLineageReq struct {
	SubjectID   uint64  `json:"subject_id"`
	Operation   string  `json:"operation"`
	Source      string  `json:"source"`
	Destination string  `json:"destination"`
	Purpose     string  `json:"purpose"`
	ConsentID   *uint64 `json:"consent_id,omitempty"`
	PayloadJSON string  `json:"payload_json"`
}

type LineageHandler struct {
	DB *sql.DB
}

func (h LineageHandler) Record(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req createLineageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.SubjectID == 0 || req.Operation == "" || req.Source == "" || req.Destination == "" || req.Purpose == "" {
		http.Error(w, "subject_id, operation, source, destination and purpose are required", http.StatusUnprocessableEntity)
		return
	}

	res, err := h.DB.ExecContext(ctx, `
		INSERT INTO data_lineage (subject_id, operation, source, destination, purpose, consent_id, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, req.SubjectID, req.Operation, req.Source, req.Destination, req.Purpose, req.ConsentID, req.PayloadJSON)
	if err != nil {
		http.Error(w, "insert error", http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":          id,
		"subject_id":  req.SubjectID,
		"operation":   req.Operation,
		"source":      req.Source,
		"destination": req.Destination,
		"purpose":     req.Purpose,
		"consent_id":  req.ConsentID,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	})
}

func (h LineageHandler) Export(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	subjectID, err := strconv.ParseUint(chi.URLParam(r, "subject_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid subject_id", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, subject_id, operation, source, destination, purpose, consent_id, payload_json, created_at
		FROM data_lineage
		WHERE subject_id = ?
		ORDER BY created_at
	`, subjectID)
	if err != nil {
		http.Error(w, "query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	events := make([]LineageEvent, 0)
	for rows.Next() {
		var e LineageEvent
		var consentID sql.NullInt64
		if err := rows.Scan(&e.ID, &e.SubjectID, &e.Operation, &e.Source, &e.Destination, &e.Purpose, &consentID, &e.PayloadJSON, &e.CreatedAt); err != nil {
			http.Error(w, "query error", http.StatusInternalServerError)
			return
		}
		if consentID.Valid {
			cid := uint64(consentID.Int64)
			e.ConsentID = &cid
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(events)
}
