package handlers

import (
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

type ProjectHandler struct {
	db  *sqlx.DB
	hub *ws.Hub
}

func NewProjectHandler(db *sqlx.DB, hub *ws.Hub) *ProjectHandler {
	return &ProjectHandler{db: db, hub: hub}
}

// GET /api/v1/projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFrom(r.Context())
	page, limit, offset := utils.PaginationParams(r)

	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("q")

	query := `SELECT * FROM projects WHERE org_id=$1`
	count := `SELECT COUNT(*) FROM projects WHERE org_id=$1`
	args := []interface{}{orgID}
	idx := 2

	if status != "" {
		query += ` AND status=$` + itoa(idx)
		count += ` AND status=$` + itoa(idx)
		args = append(args, status)
		idx++
	}
	if search != "" {
		query += ` AND (po_number ILIKE $` + itoa(idx) + ` OR client_name ILIKE $` + itoa(idx) + ` OR name ILIKE $` + itoa(idx) + `)`
		count += ` AND (po_number ILIKE $` + itoa(idx) + ` OR client_name ILIKE $` + itoa(idx) + ` OR name ILIKE $` + itoa(idx) + `)`
		args = append(args, "%"+search+"%")
		idx++
	}

	var total int
	_ = h.db.GetContext(r.Context(), &total, count, args...)

	query += ` ORDER BY created_at DESC LIMIT $` + itoa(idx) + ` OFFSET $` + itoa(idx+1)
	args = append(args, limit, offset)

	var projects []models.Project
	if err := h.db.SelectContext(r.Context(), &projects, query, args...); err != nil {
		utils.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	pages := (total + limit - 1) / limit
	utils.Paginated(w, projects, utils.PaginationMeta{Page: page, Limit: limit, Total: total, TotalPages: pages})
}

// POST /api/v1/projects
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())

	var body struct {
		PONumber        string     `json:"po_number"`
		ClientName      string     `json:"client_name"`
		ClientContact   string     `json:"client_contact"`
		Name            string     `json:"name"`
		Quantity        int        `json:"quantity"`
		Dimensions      string     `json:"dimensions"`
		Specifications  string     `json:"specifications"`
		MaterialDetails string     `json:"material_details"`
		ColorDetails    string     `json:"color_details"`
		Upholstery      string     `json:"upholstery"`
		Finish          string     `json:"finish"`
		DeliveryDate    *time.Time `json:"delivery_date"`
		DeliveryAddress string     `json:"delivery_address"`
		Remarks         string     `json:"remarks"`
		CoverImageURL   string     `json:"cover_image_url"`
		CADFilesURL     string     `json:"cad_files_url"`
		DrawingsURL     string     `json:"drawings_url"`
		JobCardsURL     string     `json:"job_cards_url"`
		RenderFilesURL  string     `json:"render_files_url"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.PONumber == "" || body.ClientName == "" || body.Name == "" {
		utils.Error(w, http.StatusBadRequest, "po_number, client_name, name are required")
		return
	}
	if body.Quantity < 1 {
		body.Quantity = 1
	}

	id := uuid.New()
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO projects(id,org_id,po_number,client_name,client_contact,name,quantity,
		 dimensions,specifications,material_details,color_details,upholstery,finish,
		 delivery_date,delivery_address,remarks,cover_image_url,cad_files_url,
		 drawings_url,job_cards_url,render_files_url,created_by)
		 VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7,
		 NULLIF($8,''),NULLIF($9,''),NULLIF($10,''),NULLIF($11,''),NULLIF($12,''),NULLIF($13,''),
		 $14,NULLIF($15,''),NULLIF($16,''),NULLIF($17,''),NULLIF($18,''),
		 NULLIF($19,''),NULLIF($20,''),NULLIF($21,''),$22)`,
		id, orgID, body.PONumber, body.ClientName, body.ClientContact, body.Name, body.Quantity,
		body.Dimensions, body.Specifications, body.MaterialDetails, body.ColorDetails,
		body.Upholstery, body.Finish, body.DeliveryDate, body.DeliveryAddress, body.Remarks,
		body.CoverImageURL, body.CADFilesURL, body.DrawingsURL, body.JobCardsURL, body.RenderFilesURL,
		userID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "create failed: "+err.Error())
		return
	}

	// notify layer 2
	h.hub.SendToOrg(orgID, string(models.NotifProjectCreated), map[string]interface{}{
		"project_id": id, "name": body.Name, "po_number": body.PONumber,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "PROJECT_CREATED", "PROJECT", &id, nil, nil)
	utils.Success(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// GET /api/v1/projects/{id}
func (h *ProjectHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	var p models.Project
	if err := h.db.GetContext(r.Context(), &p,
		`SELECT * FROM projects WHERE id=$1 AND org_id=$2`, id, orgID); err != nil {
		utils.Error(w, http.StatusNotFound, "project not found")
		return
	}
	utils.Success(w, http.StatusOK, p)
}

// PATCH /api/v1/projects/{id}   (Admin only — triggers revision)
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())

	var p models.Project
	if err := h.db.GetContext(r.Context(), &p,
		`SELECT * FROM projects WHERE id=$1 AND org_id=$2`, id, orgID); err != nil {
		utils.Error(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		PONumber        string     `json:"po_number"`
		ClientName      string     `json:"client_name"`
		ClientContact   string     `json:"client_contact"`
		Name            string     `json:"name"`
		Quantity        *int       `json:"quantity"`
		Dimensions      string     `json:"dimensions"`
		Specifications  string     `json:"specifications"`
		MaterialDetails string     `json:"material_details"`
		ColorDetails    string     `json:"color_details"`
		Upholstery      string     `json:"upholstery"`
		Finish          string     `json:"finish"`
		DeliveryDate    *time.Time `json:"delivery_date"`
		DeliveryAddress string     `json:"delivery_address"`
		Remarks         string     `json:"remarks"`
		CoverImageURL   string     `json:"cover_image_url"`
		CADFilesURL     string     `json:"cad_files_url"`
		DrawingsURL     string     `json:"drawings_url"`
		JobCardsURL     string     `json:"job_cards_url"`
		RenderFilesURL  string     `json:"render_files_url"`
		Reason          string     `json:"reason"`
		ClientRequestRef string   `json:"client_request_ref"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	prevJSON, _ := json.Marshal(p)

	newRev := p.CurrentRevision + 1
	_, err := h.db.ExecContext(r.Context(),
		`UPDATE projects SET
		 po_number=COALESCE(NULLIF($1,''),po_number),
		 client_name=COALESCE(NULLIF($2,''),client_name),
		 client_contact=COALESCE(NULLIF($3,''),client_contact),
		 name=COALESCE(NULLIF($4,''),name),
		 quantity=COALESCE($5,quantity),
		 dimensions=COALESCE(NULLIF($6,''),dimensions),
		 specifications=COALESCE(NULLIF($7,''),specifications),
		 material_details=COALESCE(NULLIF($8,''),material_details),
		 color_details=COALESCE(NULLIF($9,''),color_details),
		 upholstery=COALESCE(NULLIF($10,''),upholstery),
		 finish=COALESCE(NULLIF($11,''),finish),
		 delivery_date=COALESCE($12,delivery_date),
		 delivery_address=COALESCE(NULLIF($13,''),delivery_address),
		 remarks=COALESCE(NULLIF($14,''),remarks),
		 cover_image_url=COALESCE(NULLIF($15,''),cover_image_url),
		 cad_files_url=COALESCE(NULLIF($16,''),cad_files_url),
		 drawings_url=COALESCE(NULLIF($17,''),drawings_url),
		 job_cards_url=COALESCE(NULLIF($18,''),job_cards_url),
		 render_files_url=COALESCE(NULLIF($19,''),render_files_url),
		 current_revision=$20, updated_at=NOW()
		 WHERE id=$21 AND org_id=$22`,
		body.PONumber, body.ClientName, body.ClientContact, body.Name, body.Quantity,
		body.Dimensions, body.Specifications, body.MaterialDetails, body.ColorDetails,
		body.Upholstery, body.Finish, body.DeliveryDate, body.DeliveryAddress, body.Remarks,
		body.CoverImageURL, body.CADFilesURL, body.DrawingsURL, body.JobCardsURL, body.RenderFilesURL,
		newRev, id, orgID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "update failed")
		return
	}

	// capture revision
	var updatedP models.Project
	_ = h.db.GetContext(r.Context(), &updatedP, `SELECT * FROM projects WHERE id=$1`, id)
	newJSON, _ := json.Marshal(updatedP)

	revID := uuid.New()
	_, _ = h.db.ExecContext(r.Context(),
		`INSERT INTO project_revisions(id,project_id,revision_number,updated_by,reason,client_request_ref,prev_values,new_values)
		 VALUES($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7,$8)`,
		revID, id, newRev, userID, body.Reason, body.ClientRequestRef, prevJSON, newJSON,
	)

	h.hub.SendToOrg(orgID, string(models.NotifProjectRevision), map[string]interface{}{
		"project_id": id, "revision": newRev,
	})
	writeAudit(r.Context(), h.db, orgID, &userID, "PROJECT_UPDATED", "PROJECT", &id, prevJSON, newJSON)
	utils.SuccessMessage(w, http.StatusOK, "project updated")
}

// PATCH /api/v1/projects/{id}/status
func (h *ProjectHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())
	var body struct {
		Status models.ProjectStatus `json:"status"`
	}
	if err := utils.ParseBody(r, &body); err != nil || body.Status == "" {
		utils.Error(w, http.StatusBadRequest, "status required")
		return
	}
	_, err := h.db.ExecContext(r.Context(),
		`UPDATE projects SET status=$1, updated_at=NOW() WHERE id=$2 AND org_id=$3`,
		body.Status, id, orgID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeAudit(r.Context(), h.db, orgID, &userID, "PROJECT_STATUS_CHANGED", "PROJECT", &id, nil, nil)
	utils.SuccessMessage(w, http.StatusOK, "status updated")
}

// GET /api/v1/projects/{id}/revisions
func (h *ProjectHandler) ListRevisions(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())

	// verify project belongs to org
	var count int
	_ = h.db.GetContext(r.Context(), &count, `SELECT COUNT(*) FROM projects WHERE id=$1 AND org_id=$2`, id, orgID)
	if count == 0 {
		utils.Error(w, http.StatusNotFound, "project not found")
		return
	}
	var revs []models.ProjectRevision
	_ = h.db.SelectContext(r.Context(), &revs,
		`SELECT * FROM project_revisions WHERE project_id=$1 ORDER BY revision_number DESC`, id)
	utils.Success(w, http.StatusOK, revs)
}

// GET /api/v1/projects/{id}/timeline
func (h *ProjectHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())

	var count int
	_ = h.db.GetContext(r.Context(), &count, `SELECT COUNT(*) FROM projects WHERE id=$1 AND org_id=$2`, id, orgID)
	if count == 0 {
		utils.Error(w, http.StatusNotFound, "project not found")
		return
	}
	var logs []models.AuditLog
	_ = h.db.SelectContext(r.Context(), &logs,
		`SELECT * FROM audit_logs WHERE project_id=$1 ORDER BY created_at DESC LIMIT 200`, id)
	utils.Success(w, http.StatusOK, logs)
}

// helper
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
