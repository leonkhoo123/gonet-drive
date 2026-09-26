package controller

import (
	"net/http"
	"strconv"
	"strings"

	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/repository"

	"github.com/gin-gonic/gin"
)

const (
	defaultAuditLogPageSize = 50
	maxAuditLogPageSize     = 200
	maxAuditFilterLen       = 128
)

// RegisterAuditLogAdminRoutes wires the admin-only audit log endpoints.
func RegisterAuditLogAdminRoutes(adminRouter *gin.RouterGroup, repo repository.AuditLogRepository) {
	auditGroup := adminRouter.Group("/audit-logs")
	{
		auditGroup.GET("", listAuditLogs(repo))
	}
}

// listAuditLogs returns a filtered, paginated page of auth/security audit events.
// @Summary      List Audit Logs
// @Description  Get auth and security audit events (login, logout, MFA, session, user lifecycle). Requires admin role.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        page        query  int     false  "Page number (1-based)"
// @Param        page_size   query  int     false  "Page size (max 200)"
// @Param        event_type  query  string  false  "Filter by exact event type"
// @Param        username    query  string  false  "Filter by username (substring)"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/user/admin/audit-logs [get]
func listAuditLogs(repo repository.AuditLogRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if repo == nil {
			httpx.Err(c, http.StatusInternalServerError, "audit log repository not initialized")
			return
		}

		page := parsePositiveQueryInt(c.Query("page"), 1)
		pageSize := parsePositiveQueryInt(c.Query("page_size"), defaultAuditLogPageSize)
		if pageSize > maxAuditLogPageSize {
			pageSize = maxAuditLogPageSize
		}

		filter := repository.AuditLogFilter{
			EventType: truncateFilter(strings.TrimSpace(c.Query("event_type"))),
			Username:  truncateFilter(strings.TrimSpace(c.Query("username"))),
			Limit:     pageSize,
			Offset:    (page - 1) * pageSize,
		}

		entries, total, err := repo.List(filter)
		if err != nil {
			logger.L.Error("failed to query audit logs", "err", err)
			httpx.Err(c, http.StatusInternalServerError, "failed to query audit logs")
			return
		}

		httpx.OK(c, http.StatusOK, gin.H{
			"logs":      entries,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
	}
}

// parsePositiveQueryInt parses a query param as an int >= 1, falling back to
// def when empty, malformed, or out of range.
func parsePositiveQueryInt(raw string, def int) int {
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return def
	}
	return n
}

// truncateFilter caps user-supplied filter strings so a single request cannot
// push an unbounded value into the query.
func truncateFilter(s string) string {
	if len(s) > maxAuditFilterLen {
		return s[:maxAuditFilterLen]
	}
	return s
}
