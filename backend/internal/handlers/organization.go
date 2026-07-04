package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/oaknore/pms3/internal/middleware"
	"github.com/oaknore/pms3/internal/models"
	"github.com/oaknore/pms3/internal/utils"
)

type OrgHandler struct{ db *sqlx.DB }

func NewOrgHandler(db *sqlx.DB) *OrgHandler { return &OrgHandler{db: db} }

// GET /api/v1/org
func (h *OrgHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFrom(r.Context())
	var org models.Organization
	if err := h.db.GetContext(r.Context(), &org, `SELECT * FROM organizations WHERE id=$1`, orgID); err != nil {
		utils.Error(w, http.StatusNotFound, "organization not found")
		return
	}
	utils.Success(w, http.StatusOK, org)
}

// PATCH /api/v1/org
func (h *OrgHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFrom(r.Context())
	var body struct {
		Name    string `json:"name"`
		LogoURL string `json:"logo_url"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	_, err := h.db.ExecContext(r.Context(),
		`UPDATE organizations SET name=COALESCE(NULLIF($1,''),name),
		 logo_url=COALESCE(NULLIF($2,''),logo_url), updated_at=NOW() WHERE id=$3`,
		body.Name, body.LogoURL, orgID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "update failed")
		return
	}
	utils.SuccessMessage(w, http.StatusOK, "organization updated")
}

// ── Departments ──────────────────────────────────────────

type DeptHandler struct{ db *sqlx.DB }

func NewDeptHandler(db *sqlx.DB) *DeptHandler { return &DeptHandler{db: db} }

// GET /api/v1/departments
func (h *DeptHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFrom(r.Context())
	var depts []models.Department
	if err := h.db.SelectContext(r.Context(), &depts,
		`SELECT * FROM departments WHERE org_id=$1 ORDER BY layer, name`, orgID); err != nil {
		utils.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	utils.Success(w, http.StatusOK, depts)
}

// POST /api/v1/departments
func (h *DeptHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFrom(r.Context())
	var body struct {
		Name         string                  `json:"name"`
		Layer        models.DepartmentLayer  `json:"layer"`
		ParentDeptID *uuid.UUID              `json:"parent_dept_id"`
		Description  string                  `json:"description"`
	}
	if err := utils.ParseBody(r, &body); err != nil || body.Name == "" {
		utils.Error(w, http.StatusBadRequest, "name and layer are required")
		return
	}
	id := uuid.New()
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO departments(id,org_id,name,layer,parent_dept_id,description)
		 VALUES($1,$2,$3,$4,$5,NULLIF($6,''))`,
		id, orgID, body.Name, body.Layer, body.ParentDeptID, body.Description)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "create failed")
		return
	}
	utils.Success(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// GET /api/v1/departments/{id}
func (h *DeptHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	var dept models.Department
	if err := h.db.GetContext(r.Context(), &dept,
		`SELECT * FROM departments WHERE id=$1 AND org_id=$2`, id, orgID); err != nil {
		utils.Error(w, http.StatusNotFound, "department not found")
		return
	}
	utils.Success(w, http.StatusOK, dept)
}

// PATCH /api/v1/departments/{id}
func (h *DeptHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	_, err := h.db.ExecContext(r.Context(),
		`UPDATE departments SET
		 name=COALESCE(NULLIF($1,''),name),
		 description=COALESCE(NULLIF($2,''),description),
		 is_active=COALESCE($3,is_active),
		 updated_at=NOW()
		 WHERE id=$4 AND org_id=$5`,
		body.Name, body.Description, body.IsActive, id, orgID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "update failed")
		return
	}
	utils.SuccessMessage(w, http.StatusOK, "department updated")
}
