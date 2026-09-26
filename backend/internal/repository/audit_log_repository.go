package repository

import (
	"database/sql"
	"strings"
	"time"
)

// AuditLogFilter narrows and paginates an audit log query.
type AuditLogFilter struct {
	EventType string
	Username  string
	Limit     int
	Offset    int
}

// AuditLogEntry is a single persisted security/audit event.
type AuditLogEntry struct {
	ID         int64     `json:"id"`
	EventType  string    `json:"event_type"`
	Username   string    `json:"username"`
	IPAddress  string    `json:"ip_address"`
	DeviceInfo string    `json:"device_info"`
	FamilyID   string    `json:"family_id"`
	Metadata   string    `json:"metadata"`
	Timestamp  time.Time `json:"timestamp"`
}

// AuditLogRepository reads events written by the gonet-auth AuditLogStore.
type AuditLogRepository interface {
	// List returns a page of audit entries (newest first) and the total number
	// of rows matching the filter (ignoring pagination).
	List(filter AuditLogFilter) ([]AuditLogEntry, int, error)
}

type SQLiteAuditLogRepo struct {
	DB *sql.DB
}

func NewSQLiteAuditLogRepo(db *sql.DB) *SQLiteAuditLogRepo {
	return &SQLiteAuditLogRepo{DB: db}
}

func (r *SQLiteAuditLogRepo) List(filter AuditLogFilter) ([]AuditLogEntry, int, error) {
	where, args := buildAuditLogWhere(filter)

	var total int
	if err := r.DB.QueryRow(`SELECT COUNT(*) FROM audit_logs`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Limit/Offset are validated and bounded by the controller, so they are safe
	// to interpolate as bound parameters.
	query := `SELECT id, event_type, COALESCE(username, ''), COALESCE(ip_address, ''),
	                 COALESCE(device_info, ''), COALESCE(family_id, ''), COALESCE(metadata, ''), timestamp
	          FROM audit_logs` + where + ` ORDER BY timestamp DESC, id DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := make([]AuditLogEntry, 0)
	for rows.Next() {
		var e AuditLogEntry
		if err := rows.Scan(
			&e.ID, &e.EventType, &e.Username, &e.IPAddress,
			&e.DeviceInfo, &e.FamilyID, &e.Metadata, &e.Timestamp,
		); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

func buildAuditLogWhere(filter AuditLogFilter) (string, []any) {
	var clauses []string
	args := make([]any, 0, 2)

	if filter.EventType != "" {
		clauses = append(clauses, "event_type = ?")
		args = append(args, filter.EventType)
	}
	if filter.Username != "" {
		clauses = append(clauses, "username LIKE ?")
		args = append(args, "%"+filter.Username+"%")
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
