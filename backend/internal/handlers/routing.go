package handlers

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/oaknore/pms3/internal/middleware"
	"github.com/oaknore/pms3/internal/models"
	"github.com/oaknore/pms3/internal/utils"
	ws "github.com/oaknore/pms3/internal/websocket"
)

type RoutingHandler struct {
	db  *sqlx.DB
	hub *ws.Hub
}

func NewRoutingHandler(db *sqlx.DB, hub *ws.Hub) *RoutingHandler {
	return &RoutingHandler{db: db, hub: hub}
}

// POST /api/v1/projects/{projectID}/routings
func (h *RoutingHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())

	var body struct {
		Notes string `json:"notes"`
		Steps []struct {
			StepOrder        int                    `json:"step_order"`
			Label            string                 `json:"label"`
			DependencyPolicy models.DependencyPolicy `json:"dependency_policy"`
			DepartmentIDs    []uuid.UUID            `json:"department_ids"`
		} `json:"steps"`
		SubtaskTemplates map[string][]struct { // dept_id -> subtasks
			Title       string `json:"title"`
			Description string `json:"description"`
			IsRequired  bool   `json:"is_required"`
			SortOrder   int    `json:"sort_order"`
		} `json:"subtask_templates"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	// find next version
	var maxVer sql.NullInt64
	_ = h.db.GetContext(r.Context(), &maxVer,
		`SELECT MAX(version) FROM routings WHERE project_id=$1`, projectID)
	nextVer := int(maxVer.Int64) + 1

	// supersede previous active routing
	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE routings SET status='SUPERSEDED', updated_at=NOW() WHERE project_id=$1 AND status='ACTIVE'`, projectID)

	routingID := uuid.New()
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO routings(id,project_id,version,status,created_by,notes)
		 VALUES($1,$2,$3,'ACTIVE',$4,NULLIF($5,''))`,
		routingID, projectID, nextVer, userID, body.Notes)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "routing create failed")
		return
	}

	// insert steps + dept junctions + spawn tasks
	for _, step := range body.Steps {
		if step.DependencyPolicy == "" {
			step.DependencyPolicy = models.RequireAll
		}
		stepID := uuid.New()
		_, _ = h.db.ExecContext(r.Context(),
			`INSERT INTO routing_steps(id,routing_id,step_order,dependency_policy,label)
			 VALUES($1,$2,$3,$4,NULLIF($5,''))`,
			stepID, routingID, step.StepOrder, step.DependencyPolicy, step.Label)

		for _, deptID := range step.DepartmentIDs {
			_, _ = h.db.ExecContext(r.Context(),
				`INSERT INTO routing_step_departments(routing_step_id,department_id) VALUES($1,$2)`,
				stepID, deptID)

			// spawn task (only step 1 is PENDING active; others wait for gate)
			taskStatus := models.TaskPending
			taskID := uuid.New()
			_, _ = h.db.ExecContext(r.Context(),
				`INSERT INTO department_tasks(id,project_id,routing_id,routing_step_id,department_id,status)
				 VALUES($1,$2,$3,$4,$5,$6)`,
				taskID, projectID, routingID, stepID, deptID, taskStatus)

			// spawn subtasks from template
			if subs, ok := body.SubtaskTemplates[deptID.String()]; ok {
				for _, st := range subs {
					_, _ = h.db.ExecContext(r.Context(),
						`INSERT INTO subtasks(id,task_id,project_id,title,description,is_required,sort_order)
						 VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7)`,
						uuid.New(), taskID, projectID, st.Title, st.Description, st.IsRequired, st.SortOrder)
				}
			}
		}
	}

	// update project status to ROUTING
	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE projects SET status='ROUTING', updated_at=NOW() WHERE id=$1 AND org_id=$2`, projectID, orgID)

	h.hub.SendToOrg(orgID, string(models.NotifRoutingAssigned), map[string]interface{}{
		"project_id": projectID, "routing_id": routingID, "version": nextVer,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "ROUTING_CREATED", "ROUTING", &routingID, nil, nil)
	utils.Success(w, http.StatusCreated, map[string]interface{}{"id": routingID, "version": nextVer})
}

// GET /api/v1/projects/{projectID}/routings
func (h *RoutingHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	var routings []models.Routing
	_ = h.db.SelectContext(r.Context(), &routings,
		`SELECT * FROM routings WHERE project_id=$1 ORDER BY version DESC`, projectID)
	utils.Success(w, http.StatusOK, routings)
}

// GET /api/v1/projects/{projectID}/routings/{routingID}
func (h *RoutingHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	routingID, _ := uuid.Parse(chi.URLParam(r, "routingID"))
	var routing models.Routing
	if err := h.db.GetContext(r.Context(), &routing,
		`SELECT * FROM routings WHERE id=$1`, routingID); err != nil {
		utils.Error(w, http.StatusNotFound, "routing not found")
		return
	}

	var steps []models.RoutingStep
	_ = h.db.SelectContext(r.Context(), &steps,
		`SELECT * FROM routing_steps WHERE routing_id=$1 ORDER BY step_order`, routingID)

	for i, step := range steps {
		var deptIDs []uuid.UUID
		rows, _ := h.db.QueryContext(r.Context(),
			`SELECT department_id FROM routing_step_departments WHERE routing_step_id=$1`, step.ID)
		for rows != nil && rows.Next() {
			var id uuid.UUID
			_ = rows.Scan(&id)
			deptIDs = append(deptIDs, id)
		}
		if rows != nil {
			_ = rows.Close()
		}
		steps[i].DepartmentIDs = deptIDs
	}
	routing.Steps = steps
	utils.Success(w, http.StatusOK, routing)
}

// evaluateRoutingGate checks if the next routing step should be activated.
// Called after a task completes.
func evaluateRoutingGate(ctx context.Context, db *sqlx.DB, hub *ws.Hub, task models.DepartmentTask, orgID uuid.UUID) {
	// get current step
	var step models.RoutingStep
	if err := db.GetContext(ctx, &step,
		`SELECT * FROM routing_steps WHERE id=$1`, task.RoutingStepID); err != nil {
		return
	}

	// count tasks in this step
	var total, completed int
	_ = db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM department_tasks WHERE routing_step_id=$1`, step.ID)
	_ = db.GetContext(ctx, &completed,
		`SELECT COUNT(*) FROM department_tasks WHERE routing_step_id=$1 AND status='COMPLETED'`, step.ID)

	gate := false
	switch step.DependencyPolicy {
	case models.RequireAll:
		gate = completed == total
	case models.RequireAny:
		gate = completed >= 1
	}

	if !gate {
		return
	}

	// find next step
	var nextStep models.RoutingStep
	err := db.GetContext(ctx, &nextStep,
		`SELECT * FROM routing_steps WHERE routing_id=$1 AND step_order=$2`,
		step.RoutingID, step.StepOrder+1)
	if err != nil {
		// no next step — all routing done, mark project IN_PROGRESS→COMPLETED
		_, _ = db.ExecContext(ctx,
			`UPDATE projects SET status='COMPLETED', updated_at=NOW()
			 WHERE id=(SELECT project_id FROM routings WHERE id=$1)`, step.RoutingID)
		return
	}

	// activate tasks in next step
	rows, _ := db.QueryContext(ctx,
		`SELECT department_id FROM routing_step_departments WHERE routing_step_id=$1`, nextStep.ID)
	if rows != nil {
		for rows.Next() {
			var deptID uuid.UUID
			_ = rows.Scan(&deptID)
			// set their tasks to PENDING (they were created as PENDING, just notify)
			hub.SendToOrg(orgID, string(models.NotifTaskAssigned), map[string]interface{}{
				"routing_step_id": nextStep.ID, "department_id": deptID,
			})
		}
		_ = rows.Close()
	}
	// update project status
	_, _ = db.ExecContext(ctx,
		`UPDATE projects SET status='IN_PROGRESS', updated_at=NOW()
		 WHERE id=(SELECT project_id FROM routings WHERE id=$1)`, step.RoutingID)
}
