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

type ReworkHandler struct {
	db  *sqlx.DB
	hub *ws.Hub
}

func NewReworkHandler(db *sqlx.DB, hub *ws.Hub) *ReworkHandler {
	return &ReworkHandler{db: db, hub: hub}
}

// GET /api/v1/projects/{projectID}/reworks
func (h *ReworkHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	var reworks []models.ReworkRequest
	_ = h.db.SelectContext(r.Context(), &reworks,
		`SELECT * FROM rework_requests WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	utils.Success(w, http.StatusOK, reworks)
}

// POST /api/v1/projects/{projectID}/reworks  (Layer 3)
func (h *ReworkHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())
	deptID := middleware.DeptIDFrom(r.Context())
	role := middleware.RoleFrom(r.Context())

	if deptID == nil && role != models.RoleSuperAdmin && role != models.RoleAdmin {
		utils.Error(w, http.StatusForbidden, "must belong to a department")
		return
	}
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
		OriginatingTaskID uuid.UUID `json:"originating_task_id"`
		TargetDeptID      uuid.UUID `json:"target_dept_id"`
		Reason            string    `json:"reason"`
	}
	if err := utils.ParseBody(r, &body); err != nil || body.Reason == "" {
		utils.Error(w, http.StatusBadRequest, "originating_task_id, target_dept_id, reason required")
		return
	}

	id := uuid.New()
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO rework_requests(id,project_id,originating_task_id,requested_by,requested_dept,target_dept_id,reason)
		 VALUES($1,$2,$3,$4,$5,$6,$7)`,
		id, projectID, body.OriginatingTaskID, userID, *effectiveDept, body.TargetDeptID, body.Reason)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "create failed")
		return
	}

	h.hub.SendToOrg(orgID, string(models.NotifReworkRequest), map[string]interface{}{
		"rework_id": id, "project_id": projectID,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "REWORK_REQUESTED", "REWORK", &id, nil, nil)
	utils.Success(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// GET /api/v1/reworks/{id}
func (h *ReworkHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var rw models.ReworkRequest
	if err := h.db.GetContext(r.Context(), &rw, `SELECT * FROM rework_requests WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "rework not found")
		return
	}
	utils.Success(w, http.StatusOK, rw)
}

// PATCH /api/v1/reworks/{id}/review  (Layer 2)
func (h *ReworkHandler) Review(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())

	var body struct {
		Decision string `json:"decision"` // "approve" | "reject"
		Notes    string `json:"notes"`
		// When approving: define the new routing steps
		NewSteps []struct {
			StepOrder        int                     `json:"step_order"`
			Label            string                  `json:"label"`
			DependencyPolicy models.DependencyPolicy `json:"dependency_policy"`
			DepartmentIDs    []uuid.UUID             `json:"department_ids"`
		} `json:"new_steps"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	var rw models.ReworkRequest
	if err := h.db.GetContext(r.Context(), &rw, `SELECT * FROM rework_requests WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "rework not found")
		return
	}

	now := time.Now()
	if body.Decision == "reject" {
		_, _ = h.db.ExecContext(r.Context(),
			`UPDATE rework_requests SET status='REJECTED', reviewed_by=$1, reviewed_at=$2,
			 review_notes=NULLIF($3,''), updated_at=NOW() WHERE id=$4`,
			userID, now, body.Notes, id)
		utils.SuccessMessage(w, http.StatusOK, "rework rejected")
		return
	}

	// Approve: create a new routing version as a child of the current active routing
	var parentRoutingID uuid.UUID
	_ = h.db.GetContext(r.Context(), &parentRoutingID,
		`SELECT id FROM routings WHERE project_id=$1 AND status='ACTIVE'`, rw.ProjectID)

	var maxVer int
	_ = h.db.GetContext(r.Context(), &maxVer,
		`SELECT COALESCE(MAX(version),0) FROM routings WHERE project_id=$1`, rw.ProjectID)

	// Supersede current active
	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE routings SET status='SUPERSEDED', updated_at=NOW() WHERE project_id=$1 AND status='ACTIVE'`, rw.ProjectID)

	newRoutingID := uuid.New()
	_, _ = h.db.ExecContext(r.Context(),
		`INSERT INTO routings(id,project_id,version,parent_routing_id,status,created_by,notes)
		 VALUES($1,$2,$3,$4,'ACTIVE',$5,$6)`,
		newRoutingID, rw.ProjectID, maxVer+1, parentRoutingID, userID,
		"Rework routing — from rework request "+id.String())

	// Build new steps
	for _, step := range body.NewSteps {
		if step.DependencyPolicy == "" {
			step.DependencyPolicy = models.RequireAll
		}
		stepID := uuid.New()
		_, _ = h.db.ExecContext(r.Context(),
			`INSERT INTO routing_steps(id,routing_id,step_order,dependency_policy,label)
			 VALUES($1,$2,$3,$4,NULLIF($5,''))`,
			stepID, newRoutingID, step.StepOrder, step.DependencyPolicy, step.Label)
		for _, deptID := range step.DepartmentIDs {
			_, _ = h.db.ExecContext(r.Context(),
				`INSERT INTO routing_step_departments(routing_step_id,department_id) VALUES($1,$2)`, stepID, deptID)
			taskID := uuid.New()
			_, _ = h.db.ExecContext(r.Context(),
				`INSERT INTO department_tasks(id,project_id,routing_id,routing_step_id,department_id,status)
				 VALUES($1,$2,$3,$4,$5,'PENDING')`,
				taskID, rw.ProjectID, newRoutingID, stepID, deptID)
		}
	}

	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE rework_requests SET status='APPROVED', reviewed_by=$1, reviewed_at=$2,
		 review_notes=NULLIF($3,''), new_routing_id=$4, updated_at=NOW() WHERE id=$5`,
		userID, now, body.Notes, newRoutingID, id)

	h.hub.SendToOrg(orgID, string(models.NotifDeptReopened), map[string]interface{}{
		"project_id": rw.ProjectID, "rework_id": id, "new_routing_id": newRoutingID,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "REWORK_APPROVED", "REWORK", &id, nil, nil)
	utils.Success(w, http.StatusOK, map[string]interface{}{
		"message": "rework approved", "new_routing_id": newRoutingID,
	})
}
