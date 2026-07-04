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

type TaskHandler struct {
	db  *sqlx.DB
	hub *ws.Hub
}

func NewTaskHandler(db *sqlx.DB, hub *ws.Hub) *TaskHandler {
	return &TaskHandler{db: db, hub: hub}
}

// GET /api/v1/projects/{projectID}/tasks
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	deptFilter := r.URL.Query().Get("department_id")

	query := `SELECT * FROM department_tasks WHERE project_id=$1`
	args := []interface{}{projectID}
	if deptFilter != "" {
		query += ` AND department_id=$2`
		args = append(args, deptFilter)
	}
	query += ` ORDER BY created_at`

	var tasks []models.DepartmentTask
	if err := h.db.SelectContext(r.Context(), &tasks, query, args...); err != nil {
		utils.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	utils.Success(w, http.StatusOK, tasks)
}

// GET /api/v1/tasks/{id}
func (h *TaskHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var task models.DepartmentTask
	if err := h.db.GetContext(r.Context(), &task,
		`SELECT * FROM department_tasks WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "task not found")
		return
	}

	// load assigned users
	var users []models.User
	_ = h.db.SelectContext(r.Context(), &users,
		`SELECT u.* FROM users u
		 JOIN task_assignments ta ON ta.user_id=u.id
		 WHERE ta.task_id=$1`, id)
	task.AssignedUsers = users

	// load subtasks
	var subtasks []models.Subtask
	_ = h.db.SelectContext(r.Context(), &subtasks,
		`SELECT * FROM subtasks WHERE task_id=$1 ORDER BY sort_order`, id)
	task.Subtasks = subtasks

	utils.Success(w, http.StatusOK, task)
}

// PATCH /api/v1/tasks/{id}
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())

	var task models.DepartmentTask
	if err := h.db.GetContext(r.Context(), &task, `SELECT * FROM department_tasks WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "task not found")
		return
	}

	var body struct {
		StartDate   *time.Time   `json:"start_date"`
		DueDate     *time.Time   `json:"due_date"`
		Priority    *int         `json:"priority"`
		Notes       string       `json:"notes"`
		AssigneeIDs []uuid.UUID  `json:"assignee_ids"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	// dates freeze after first save
	if task.DatesFrozen && (body.StartDate != nil || body.DueDate != nil) {
		utils.Error(w, http.StatusBadRequest, "task dates are frozen and cannot be changed directly")
		return
	}

	freeze := body.StartDate != nil || body.DueDate != nil || task.DatesFrozen
	prevJSON, _ := json.Marshal(task)

	_, err := h.db.ExecContext(r.Context(),
		`UPDATE department_tasks SET
		 start_date=COALESCE($1,start_date),
		 due_date=COALESCE($2,due_date),
		 priority=COALESCE($3,priority),
		 notes=COALESCE(NULLIF($4,''),notes),
		 dates_frozen=$5,
		 updated_at=NOW()
		 WHERE id=$6`,
		body.StartDate, body.DueDate, body.Priority, body.Notes, freeze, id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "update failed")
		return
	}

	// sync assignees
	if body.AssigneeIDs != nil {
		_, _ = h.db.ExecContext(r.Context(), `DELETE FROM task_assignments WHERE task_id=$1`, id)
		for _, uid := range body.AssigneeIDs {
			_, _ = h.db.ExecContext(r.Context(),
				`INSERT INTO task_assignments(task_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, uid)
		}
	}

	writeAudit(r.Context(), h.db, orgID, &userID, "TASK_UPDATED", "TASK", &id, prevJSON, nil)
	utils.SuccessMessage(w, http.StatusOK, "task updated")
}

// PATCH /api/v1/tasks/{id}/status
func (h *TaskHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())

	var body struct {
		Status models.TaskStatus `json:"status"`
	}
	if err := utils.ParseBody(r, &body); err != nil || body.Status == "" {
		utils.Error(w, http.StatusBadRequest, "status required")
		return
	}

	var task models.DepartmentTask
	if err := h.db.GetContext(r.Context(), &task, `SELECT * FROM department_tasks WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "task not found")
		return
	}

	// Guard: can't complete if required subtasks are pending
	if body.Status == models.TaskCompleted {
		var pendingRequired int
		_ = h.db.GetContext(r.Context(), &pendingRequired,
			`SELECT COUNT(*) FROM subtasks WHERE task_id=$1 AND is_required=true AND status!='COMPLETED'`, id)
		if pendingRequired > 0 {
			utils.Error(w, http.StatusBadRequest, "all required subtasks must be completed first")
			return
		}
	}

	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE department_tasks SET status=$1, updated_at=NOW() WHERE id=$2`, body.Status, id)

	notifType := string(models.NotifTaskStarted)
	if body.Status == models.TaskCompleted {
		notifType = string(models.NotifTaskCompleted)
		// evaluate routing gate
		go evaluateRoutingGate(context.Background(), h.db, h.hub, task, orgID)
	}

	h.hub.SendToOrg(orgID, notifType, map[string]interface{}{
		"task_id": id, "project_id": task.ProjectID, "status": body.Status,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "TASK_STATUS_CHANGED", "TASK", &id, nil, nil)
	utils.SuccessMessage(w, http.StatusOK, "status updated")
}

// ── Subtask handlers ──────────────────────────────────────

type SubtaskHandler struct {
	db  *sqlx.DB
	hub *ws.Hub
}

func NewSubtaskHandler(db *sqlx.DB, hub *ws.Hub) *SubtaskHandler {
	return &SubtaskHandler{db: db, hub: hub}
}

// POST /api/v1/tasks/{taskID}/subtasks
func (h *SubtaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	taskID, _ := uuid.Parse(chi.URLParam(r, "taskID"))

	var task models.DepartmentTask
	if err := h.db.GetContext(r.Context(), &task, `SELECT * FROM department_tasks WHERE id=$1`, taskID); err != nil {
		utils.Error(w, http.StatusNotFound, "task not found")
		return
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		IsRequired  bool   `json:"is_required"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := utils.ParseBody(r, &body); err != nil || body.Title == "" {
		utils.Error(w, http.StatusBadRequest, "title required")
		return
	}

	id := uuid.New()
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO subtasks(id,task_id,project_id,title,description,is_required,sort_order)
		 VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7)`,
		id, taskID, task.ProjectID, body.Title, body.Description, body.IsRequired, body.SortOrder)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "create failed")
		return
	}
	utils.Success(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// PATCH /api/v1/subtasks/{id}
func (h *SubtaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var body struct {
		Title       string     `json:"title"`
		Description string     `json:"description"`
		AssignedTo  *uuid.UUID `json:"assigned_to"`
		SortOrder   *int       `json:"sort_order"`
		Notes       string     `json:"notes"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	_, err := h.db.ExecContext(r.Context(),
		`UPDATE subtasks SET
		 title=COALESCE(NULLIF($1,''),title),
		 description=COALESCE(NULLIF($2,''),description),
		 assigned_to=COALESCE($3,assigned_to),
		 sort_order=COALESCE($4,sort_order),
		 notes=COALESCE(NULLIF($5,''),notes),
		 updated_at=NOW()
		 WHERE id=$6`,
		body.Title, body.Description, body.AssignedTo, body.SortOrder, body.Notes, id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "update failed")
		return
	}
	utils.SuccessMessage(w, http.StatusOK, "subtask updated")
}

// PATCH /api/v1/subtasks/{id}/complete
func (h *SubtaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())

	var sub models.Subtask
	if err := h.db.GetContext(r.Context(), &sub, `SELECT * FROM subtasks WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "subtask not found")
		return
	}

	now := time.Now()
	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE subtasks SET status='COMPLETED', completed_at=$1, completed_by=$2, updated_at=NOW() WHERE id=$3`,
		now, userID, id)

	h.hub.SendToOrg(orgID, string(models.NotifSubtaskCompleted), map[string]interface{}{
		"subtask_id": id, "task_id": sub.TaskID, "project_id": sub.ProjectID,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "SUBTASK_COMPLETED", "SUBTASK", &id, nil, nil)
	utils.SuccessMessage(w, http.StatusOK, "subtask completed")
}

// DELETE /api/v1/subtasks/{id}
func (h *SubtaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	_, _ = h.db.ExecContext(r.Context(), `DELETE FROM subtasks WHERE id=$1`, id)
	utils.SuccessMessage(w, http.StatusOK, "subtask deleted")
}
