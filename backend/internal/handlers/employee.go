package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/oaknore/pms3/internal/middleware"
	"github.com/oaknore/pms3/internal/models"
	"github.com/oaknore/pms3/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type EmployeeHandler struct{ db *sqlx.DB }

func NewEmployeeHandler(db *sqlx.DB) *EmployeeHandler { return &EmployeeHandler{db: db} }

// GET /api/v1/employees
func (h *EmployeeHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFrom(r.Context())
	page, limit, offset := utils.PaginationParams(r)
	deptFilter := r.URL.Query().Get("department_id")

	var users []models.User
	var total int

	query := `SELECT * FROM users WHERE org_id=$1`
	countQuery := `SELECT COUNT(*) FROM users WHERE org_id=$1`
	args := []interface{}{orgID}

	if deptFilter != "" {
		query += ` AND department_id=$2 ORDER BY first_name LIMIT $3 OFFSET $4`
		countQuery += ` AND department_id=$2`
		args = append(args, deptFilter)
		_ = h.db.GetContext(r.Context(), &total, countQuery, args...)
		args = append(args, limit, offset)
	} else {
		query += ` ORDER BY first_name LIMIT $2 OFFSET $3`
		_ = h.db.GetContext(r.Context(), &total, countQuery, orgID)
		args = append(args, limit, offset)
	}

	if err := h.db.SelectContext(r.Context(), &users, query, args...); err != nil {
		utils.Error(w, http.StatusInternalServerError, "query failed")
		return
	}

	pages := total / limit
	if total%limit != 0 {
		pages++
	}
	utils.Paginated(w, users, utils.PaginationMeta{Page: page, Limit: limit, Total: total, TotalPages: pages})
}

// POST /api/v1/employees
func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFrom(r.Context())
	var body struct {
		EmployeeID   *string          `json:"employee_id"`
		FirstName    string           `json:"first_name"`
		LastName     string           `json:"last_name"`
		Email        string           `json:"email"`
		Phone        *string          `json:"phone"`
		Password     string           `json:"password"`
		Role         models.UserRole  `json:"role"`
		DepartmentID *uuid.UUID       `json:"department_id"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.FirstName == "" || body.Email == "" || body.Password == "" {
		utils.Error(w, http.StatusBadRequest, "first_name, email, password required")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "password hashing failed")
		return
	}
	id := uuid.New()
	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO users(id,org_id,employee_id,first_name,last_name,email,phone,password_hash,role,department_id)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id, orgID, body.EmployeeID, body.FirstName, body.LastName,
		body.Email, body.Phone, string(hash), body.Role, body.DepartmentID)
	if err != nil {
		utils.Error(w, http.StatusConflict, "email already exists or invalid data")
		return
	}
	utils.Success(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// GET /api/v1/employees/{id}
func (h *EmployeeHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	var user models.User
	if err := h.db.GetContext(r.Context(), &user,
		`SELECT * FROM users WHERE id=$1 AND org_id=$2`, id, orgID); err != nil {
		utils.Error(w, http.StatusNotFound, "employee not found")
		return
	}
	utils.Success(w, http.StatusOK, user)
}

// PATCH /api/v1/employees/{id}
func (h *EmployeeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	var body struct {
		FirstName    *string    `json:"first_name"`
		LastName     *string    `json:"last_name"`
		Phone        *string    `json:"phone"`
		DepartmentID *uuid.UUID `json:"department_id"`
		IsActive     *bool      `json:"is_active"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	_, err := h.db.ExecContext(r.Context(),
		`UPDATE users SET
		 first_name=COALESCE($1,first_name),
		 last_name=COALESCE($2,last_name),
		 phone=COALESCE($3,phone),
		 department_id=COALESCE($4,department_id),
		 is_active=COALESCE($5,is_active),
		 updated_at=NOW()
		 WHERE id=$6 AND org_id=$7`,
		body.FirstName, body.LastName, body.Phone, body.DepartmentID, body.IsActive, id, orgID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "update failed")
		return
	}
	utils.SuccessMessage(w, http.StatusOK, "employee updated")
}

// POST /api/v1/employees/{id}/reset-password   (admin only)
func (h *EmployeeHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	var body struct {
		NewPassword string `json:"new_password"`
	}
	if err := utils.ParseBody(r, &body); err != nil || len(body.NewPassword) < 8 {
		utils.Error(w, http.StatusBadRequest, "new_password (min 8 chars) required")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	_, err := h.db.ExecContext(r.Context(),
		`UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2 AND org_id=$3`,
		string(hash), id, orgID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "reset failed")
		return
	}
	utils.SuccessMessage(w, http.StatusOK, "password reset")
}

// PATCH /api/v1/employees/{id}/transfer
func (h *EmployeeHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	var body struct {
		DepartmentID uuid.UUID `json:"department_id"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "department_id required")
		return
	}
	_, err := h.db.ExecContext(r.Context(),
		`UPDATE users SET department_id=$1, updated_at=NOW() WHERE id=$2 AND org_id=$3`,
		body.DepartmentID, id, orgID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "transfer failed")
		return
	}
	utils.SuccessMessage(w, http.StatusOK, "employee transferred")
}
