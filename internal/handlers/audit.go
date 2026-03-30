package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type AuditEvent struct {
	ID          uint64 `json:"id"`
	EventType   string `json:"event_type"`
	EntityType  string `json:"entity_type"`
	EntityID    uint64 `json:"entity_id"`
	PayloadJSON string `json:"payload"`
	CreatedAt   string `json:"created_at"`
}

type AuditHandler struct {
	DB *sql.DB
}

func (h AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, event_type, entity_type, entity_id, payload_json, created_at
		FROM audit_events
		ORDER BY id
	`)
	if err != nil {
		http.Error(w, "db unreachable", http.StatusServiceUnavailable)
		return
	}
	defer rows.Close()

	events := make([]AuditEvent, 0)
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.EntityType, &e.EntityID, &e.PayloadJSON, &e.CreatedAt); err != nil {
			http.Error(w, "query error", http.StatusInternalServerError)
			return
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
