package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/oaknore/pms3/internal/middleware"
	"github.com/oaknore/pms3/internal/models"
	"github.com/oaknore/pms3/internal/utils"
	ws "github.com/oaknore/pms3/internal/websocket"
)

type ReportHandler struct {
	db  *sqlx.DB
	hub *ws.Hub
}

func NewReportHandler(db *sqlx.DB, hub *ws.Hub) *ReportHandler {
	return &ReportHandler{db: db, hub: hub}
}

// GET /api/v1/projects/{projectID}/reports
func (h *ReportHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	deptFilter := r.URL.Query().Get("department_id")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	query := `SELECT * FROM daily_reports WHERE project_id=$1`
	args := []interface{}{projectID}
	idx := 2

	if deptFilter != "" {
		query += ` AND department_id=$` + itoa(idx)
		args = append(args, deptFilter)
		idx++
	}
	if from != "" {
		query += ` AND report_date >= $` + itoa(idx)
		args = append(args, from)
		idx++
	}
	if to != "" {
		query += ` AND report_date <= $` + itoa(idx)
		args = append(args, to)
		idx++
	}
	query += ` ORDER BY report_date DESC`

	var reports []models.DailyReport
	if err := h.db.SelectContext(r.Context(), &reports, query, args...); err != nil {
		utils.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	utils.Success(w, http.StatusOK, reports)
}

// POST /api/v1/projects/{projectID}/reports
func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())
	deptID := middleware.DeptIDFrom(r.Context())

	if deptID == nil {
		utils.Error(w, http.StatusForbidden, "must belong to a department")
		return
	}

	var body struct {
		Description string     `json:"description"`
		ReportDate  *time.Time `json:"report_date"`
	}
	if err := utils.ParseBody(r, &body); err != nil || body.Description == "" {
		utils.Error(w, http.StatusBadRequest, "description required")
		return
	}

	reportDate := time.Now()
	if body.ReportDate != nil {
		reportDate = *body.ReportDate
	}

	id := uuid.New()
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO daily_reports(id,project_id,department_id,submitted_by,report_date,description)
		 VALUES($1,$2,$3,$4,$5,$6)`,
		id, projectID, *deptID, userID, reportDate.Format("2006-01-02"), body.Description)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "create failed")
		return
	}

	h.hub.SendToOrg(orgID, string(models.NotifDailyReportSubmitted), map[string]interface{}{
		"report_id": id, "project_id": projectID, "dept_id": *deptID,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "DAILY_REPORT_SUBMITTED", "DAILY_REPORT", &id, nil, nil)
	utils.Success(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// GET /api/v1/reports/{id}
func (h *ReportHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var rep models.DailyReport
	if err := h.db.GetContext(r.Context(), &rep,
		`SELECT * FROM daily_reports WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "report not found")
		return
	}
	var files []models.FileAsset
	_ = h.db.SelectContext(r.Context(), &files,
		`SELECT * FROM file_assets WHERE owner_type='DAILY_REPORT' AND owner_id=$1`, id)
	rep.Attachments = files
	utils.Success(w, http.StatusOK, rep)
}

// ── Notification handlers ─────────────────────────────────

type NotificationHandler struct{ db *sqlx.DB }

func NewNotificationHandler(db *sqlx.DB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

// GET /api/v1/notifications
func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r.Context())
	page, limit, offset := utils.PaginationParams(r)
	unreadOnly := r.URL.Query().Get("unread") == "true"

	query := `SELECT * FROM notifications WHERE recipient_id=$1`
	args := []interface{}{userID}
	if unreadOnly {
		query += ` AND is_read=false`
	}
	query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	args = append(args, limit, offset)

	var notifs []models.Notification
	_ = h.db.SelectContext(r.Context(), &notifs, query, args...)

	var total int
	_ = h.db.GetContext(r.Context(), &total,
		`SELECT COUNT(*) FROM notifications WHERE recipient_id=$1`, userID)

	pages := (total + limit - 1) / limit
	utils.Paginated(w, notifs, utils.PaginationMeta{Page: page, Limit: limit, Total: total, TotalPages: pages})
}

// PATCH /api/v1/notifications/{id}/read
func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	userID := middleware.UserIDFrom(r.Context())
	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE notifications SET is_read=true WHERE id=$1 AND recipient_id=$2`, id, userID)
	utils.SuccessMessage(w, http.StatusOK, "marked as read")
}

// PATCH /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r.Context())
	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE notifications SET is_read=true WHERE recipient_id=$1 AND is_read=false`, userID)
	utils.SuccessMessage(w, http.StatusOK, "all marked as read")
}

// GET /api/v1/notifications/unread-count
func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r.Context())
	var count int
	_ = h.db.GetContext(r.Context(), &count,
		`SELECT COUNT(*) FROM notifications WHERE recipient_id=$1 AND is_read=false`, userID)
	utils.Success(w, http.StatusOK, map[string]int{"count": count})
}

// ── Audit handler ─────────────────────────────────────────

type AuditHandler struct{ db *sqlx.DB }

func NewAuditHandler(db *sqlx.DB) *AuditHandler { return &AuditHandler{db: db} }

// GET /api/v1/audit?project_id=X&entity_type=Y&limit=50
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFrom(r.Context())
	_, limit, offset := utils.PaginationParams(r)
	projectIDStr := r.URL.Query().Get("project_id")
	entityType := r.URL.Query().Get("entity_type")

	query := `SELECT * FROM audit_logs WHERE org_id=$1`
	args := []interface{}{orgID}
	idx := 2

	if projectIDStr != "" {
		query += ` AND project_id=$` + itoa(idx)
		args = append(args, projectIDStr)
		idx++
	}
	if entityType != "" {
		query += ` AND entity_type=$` + itoa(idx)
		args = append(args, entityType)
		idx++
	}
	query += ` ORDER BY created_at DESC LIMIT $` + itoa(idx) + ` OFFSET $` + itoa(idx+1)
	args = append(args, limit, offset)

	var logs []models.AuditLog
	_ = h.db.SelectContext(r.Context(), &logs, query, args...)
	utils.Success(w, http.StatusOK, logs)
}

// ── shared audit writer used by all handlers ──────────────

func writeAudit(
	ctx context.Context,
	db *sqlx.DB,
	orgID uuid.UUID,
	actorID *uuid.UUID,
	action, entityType string,
	entityID *uuid.UUID,
	prevState, newState []byte,
) {
	meta, _ := json.Marshal(map[string]string{"source": "api"})
	_, _ = db.ExecContext(ctx,
		`INSERT INTO audit_logs(id,org_id,project_id,actor_id,action,entity_type,entity_id,prev_state,new_state,metadata)
		 VALUES($1,$2,
		   (SELECT project_id FROM (
		     SELECT project_id FROM department_tasks WHERE id=$3
		     UNION SELECT id as project_id FROM projects WHERE id=$3
		     UNION SELECT project_id FROM issues WHERE id=$3
		     UNION SELECT project_id FROM routings WHERE id=$3
		   ) sub LIMIT 1),
		 $4,$5,$6,$7,$8,$9,$10)`,
		uuid.New(), orgID, entityID, actorID, action, entityType, entityID, prevState, newState, meta,
	)
}
