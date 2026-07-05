package handlers

import (
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

type IssueHandler struct {
	db  *sqlx.DB
	hub *ws.Hub
}

func NewIssueHandler(db *sqlx.DB, hub *ws.Hub) *IssueHandler {
	return &IssueHandler{db: db, hub: hub}
}

// GET /api/v1/projects/{projectID}/issues
func (h *IssueHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	status := r.URL.Query().Get("status")

	query := `SELECT * FROM issues WHERE project_id=$1`
	args := []interface{}{projectID}
	if status != "" {
		query += ` AND status=$2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	var issues []models.Issue
	if err := h.db.SelectContext(r.Context(), &issues, query, args...); err != nil {
		utils.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	utils.Success(w, http.StatusOK, issues)
}

// POST /api/v1/projects/{projectID}/issues
func (h *IssueHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())
	deptID := middleware.DeptIDFrom(r.Context())
	role := middleware.RoleFrom(r.Context())

	// SUPER_ADMIN and ADMIN can raise issues on behalf of any department
	if deptID == nil && role != models.RoleSuperAdmin && role != models.RoleAdmin {
		utils.Error(w, http.StatusForbidden, "must belong to a department to raise issues")
		return
	}

	// Use a placeholder dept for admins with no department
	effectiveDept := deptID
	if effectiveDept == nil {
		var anyDept struct{ ID string `db:"id"` }
		_ = h.db.GetContext(r.Context(), &anyDept, `SELECT id FROM departments WHERE org_id=$1 LIMIT 1`, orgID)
		if anyDept.ID != "" {
			id, _ := uuid.Parse(anyDept.ID)
			effectiveDept = &id
		}
	}

	var body struct {
		TaskID       *uuid.UUID       `json:"task_id"`
		IssueType    models.IssueType `json:"issue_type"`
		CustomType   string           `json:"custom_type"`
		Title        string           `json:"title"`
		Description  string           `json:"description"`
		AssignedDept *uuid.UUID       `json:"assigned_dept"`
	}
	if err := utils.ParseBody(r, &body); err != nil || body.Title == "" || body.Description == "" {
		utils.Error(w, http.StatusBadRequest, "title, description, issue_type required")
		return
	}

	id := uuid.New()
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO issues(id,project_id,task_id,raised_by_dept,raised_by,issue_type,custom_type,title,description,assigned_dept)
		 VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10)`,
		id, projectID, body.TaskID, *effectiveDept, userID, body.IssueType, body.CustomType,
		body.Title, body.Description, body.AssignedDept)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "create failed: "+err.Error())
		return
	}

	// put task on ISSUE_HOLD if linked
	if body.TaskID != nil {
		_, _ = h.db.ExecContext(r.Context(),
			`UPDATE department_tasks SET status='ISSUE_HOLD', updated_at=NOW() WHERE id=$1`, *body.TaskID)
	}

	h.hub.SendToOrg(orgID, string(models.NotifIssueRaised), map[string]interface{}{
		"issue_id": id, "project_id": projectID, "title": body.Title,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "ISSUE_RAISED", "ISSUE", &id, nil, nil)
	utils.Success(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// GET /api/v1/issues/{id}
func (h *IssueHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var issue models.Issue
	if err := h.db.GetContext(r.Context(), &issue, `SELECT * FROM issues WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "issue not found")
		return
	}
	var files []models.FileAsset
	_ = h.db.SelectContext(r.Context(), &files,
		`SELECT * FROM file_assets WHERE owner_type='ISSUE' AND owner_id=$1`, id)
	issue.Attachments = files
	utils.Success(w, http.StatusOK, issue)
}

// PATCH /api/v1/issues/{id}/review  (Layer 2 — approve or reject)
func (h *IssueHandler) Review(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())

	var body struct {
		Decision string `json:"decision"` // "approve" | "reject"
		Notes    string `json:"notes"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	var issue models.Issue
	if err := h.db.GetContext(r.Context(), &issue, `SELECT * FROM issues WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "issue not found")
		return
	}

	now := time.Now()
	newStatus := models.IssueApproved
	if body.Decision == "reject" {
		newStatus = models.IssueRejected
	}

	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE issues SET status=$1, reviewed_by=$2, reviewed_at=$3, review_notes=NULLIF($4,''), updated_at=NOW()
		 WHERE id=$5`,
		newStatus, userID, now, body.Notes, id)

	// if rejected and task was on ISSUE_HOLD, revert to IN_PROGRESS
	if body.Decision == "reject" && issue.TaskID != nil {
		_, _ = h.db.ExecContext(r.Context(),
			`UPDATE department_tasks SET status='IN_PROGRESS', updated_at=NOW() WHERE id=$1`, *issue.TaskID)
		_, _ = h.db.ExecContext(r.Context(),
			`UPDATE issues SET status='CLOSED', updated_at=NOW() WHERE id=$1`, id)
	}

	notifType := string(models.NotifIssueApproved)
	if body.Decision == "reject" {
		notifType = string(models.NotifIssueClosed)
	}
	h.hub.SendToUser(issue.RaisedBy, notifType, map[string]interface{}{
		"issue_id": id, "decision": body.Decision,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "ISSUE_REVIEWED", "ISSUE", &id, nil, nil)
	utils.SuccessMessage(w, http.StatusOK, "issue reviewed")
}

// PATCH /api/v1/issues/{id}/resolve  (Layer 3)
func (h *IssueHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())

	var body struct {
		ResolutionNote string `json:"resolution_note"`
	}
	_ = utils.ParseBody(r, &body)

	var issue models.Issue
	if err := h.db.GetContext(r.Context(), &issue, `SELECT * FROM issues WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "issue not found")
		return
	}
	if issue.Status != models.IssueApproved {
		utils.Error(w, http.StatusBadRequest, "can only resolve approved issues")
		return
	}

	now := time.Now()
	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE issues SET status='RESOLVED', resolved_by=$1, resolved_at=$2,
		 resolution_note=NULLIF($3,''), updated_at=NOW() WHERE id=$4`,
		userID, now, body.ResolutionNote, id)

	// revert task from ISSUE_HOLD
	if issue.TaskID != nil {
		_, _ = h.db.ExecContext(r.Context(),
			`UPDATE department_tasks SET status='IN_PROGRESS', updated_at=NOW() WHERE id=$1`, *issue.TaskID)
	}

	h.hub.SendToOrg(orgID, string(models.NotifIssueClosed), map[string]interface{}{
		"issue_id": id, "project_id": issue.ProjectID,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "ISSUE_RESOLVED", "ISSUE", &id, nil, nil)
	utils.SuccessMessage(w, http.StatusOK, "issue resolved")
}
