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

type MaterialHandler struct {
	db  *sqlx.DB
	hub *ws.Hub
}

func NewMaterialHandler(db *sqlx.DB, hub *ws.Hub) *MaterialHandler {
	return &MaterialHandler{db: db, hub: hub}
}

// GET /api/v1/projects/{projectID}/materials
func (h *MaterialHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	var reqs []models.MaterialRequisition
	_ = h.db.SelectContext(r.Context(), &reqs,
		`SELECT * FROM material_requisitions WHERE project_id=$1 ORDER BY created_at DESC`, projectID)

	for i, req := range reqs {
		var items []models.MaterialItem
		_ = h.db.SelectContext(r.Context(), &items,
			`SELECT * FROM material_items WHERE requisition_id=$1`, req.ID)
		reqs[i].Items = items
	}
	utils.Success(w, http.StatusOK, reqs)
}

// POST /api/v1/projects/{projectID}/materials
func (h *MaterialHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())
	deptID := middleware.DeptIDFrom(r.Context())

	if deptID == nil {
		utils.Error(w, http.StatusForbidden, "must belong to a department")
		return
	}

	var body struct {
		TaskID *uuid.UUID `json:"task_id"`
		Notes  string     `json:"notes"`
		Items  []struct {
			MaterialName string  `json:"material_name"`
			Quantity     float64 `json:"quantity"`
			Unit         string  `json:"unit"`
			Description  string  `json:"description"`
		} `json:"items"`
	}
	if err := utils.ParseBody(r, &body); err != nil || len(body.Items) == 0 {
		utils.Error(w, http.StatusBadRequest, "items required")
		return
	}

	id := uuid.New()
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO material_requisitions(id,project_id,task_id,requested_by,dept_id,notes)
		 VALUES($1,$2,$3,$4,$5,NULLIF($6,''))`,
		id, projectID, body.TaskID, userID, *deptID, body.Notes)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "create failed")
		return
	}

	for _, item := range body.Items {
		_, _ = h.db.ExecContext(r.Context(),
			`INSERT INTO material_items(id,requisition_id,material_name,quantity,unit,description)
			 VALUES($1,$2,$3,$4,$5,$6)`,
			uuid.New(), id, item.MaterialName, item.Quantity, item.Unit, item.Description)
	}

	h.hub.SendToOrg(orgID, string(models.NotifMaterialRequest), map[string]interface{}{
		"requisition_id": id, "project_id": projectID,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "MATERIAL_REQUESTED", "MATERIAL_REQUISITION", &id, nil, nil)
	utils.Success(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// GET /api/v1/materials/{id}
func (h *MaterialHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req models.MaterialRequisition
	if err := h.db.GetContext(r.Context(), &req,
		`SELECT * FROM material_requisitions WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "requisition not found")
		return
	}
	var items []models.MaterialItem
	_ = h.db.SelectContext(r.Context(), &items,
		`SELECT * FROM material_items WHERE requisition_id=$1`, id)
	req.Items = items
	utils.Success(w, http.StatusOK, req)
}

// PATCH /api/v1/materials/{id}/review  (Layer 2 / Admin)
func (h *MaterialHandler) Review(w http.ResponseWriter, r *http.Request) {
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

	newStatus := models.MatReqApproved
	if body.Decision == "reject" {
		newStatus = models.MatReqRejected
	}

	now := time.Now()
	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE material_requisitions SET status=$1, reviewed_by=$2, reviewed_at=$3,
		 review_notes=NULLIF($4,''), updated_at=NOW() WHERE id=$5`,
		newStatus, userID, now, body.Notes, id)

	writeAudit(r.Context(), h.db, orgID, &userID, "MATERIAL_REVIEWED", "MATERIAL_REQUISITION", &id, nil, nil)
	utils.SuccessMessage(w, http.StatusOK, "requisition reviewed")
}

// PATCH /api/v1/materials/{id}/fulfill
func (h *MaterialHandler) Fulfill(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE material_requisitions SET status='FULFILLED', updated_at=NOW() WHERE id=$1`, id)
	utils.SuccessMessage(w, http.StatusOK, "requisition fulfilled")
}
