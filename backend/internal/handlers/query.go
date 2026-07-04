package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/oaknore/pms3/internal/middleware"
	"github.com/oaknore/pms3/internal/models"
	"github.com/oaknore/pms3/internal/utils"
	ws "github.com/oaknore/pms3/internal/websocket"
)

type QueryHandler struct {
	db  *sqlx.DB
	hub *ws.Hub
}

func NewQueryHandler(db *sqlx.DB, hub *ws.Hub) *QueryHandler {
	return &QueryHandler{db: db, hub: hub}
}

// GET /api/v1/projects/{projectID}/queries
func (h *QueryHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	userID := middleware.UserIDFrom(r.Context())

	var queries []models.Query
	_ = h.db.SelectContext(r.Context(), &queries,
		`SELECT * FROM queries
		 WHERE project_id=$1 AND (sender_id=$2 OR receiver_id=$2)
		 ORDER BY created_at DESC`,
		projectID, userID)
	utils.Success(w, http.StatusOK, queries)
}

// POST /api/v1/projects/{projectID}/queries
func (h *QueryHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, _ := uuid.Parse(chi.URLParam(r, "projectID"))
	orgID := middleware.OrgIDFrom(r.Context())
	senderID := middleware.UserIDFrom(r.Context())

	var body struct {
		ReceiverID uuid.UUID `json:"receiver_id"`
		Subject    string    `json:"subject"`
		Message    string    `json:"message"`
	}
	if err := utils.ParseBody(r, &body); err != nil || body.Subject == "" || body.Message == "" {
		utils.Error(w, http.StatusBadRequest, "receiver_id, subject, message required")
		return
	}

	// Enforce adjacent-layer-only rule server-side
	var sender, receiver models.User
	if err := h.db.GetContext(r.Context(), &sender, `SELECT * FROM users WHERE id=$1`, senderID); err != nil {
		utils.Error(w, http.StatusInternalServerError, "sender not found")
		return
	}
	if err := h.db.GetContext(r.Context(), &receiver, `SELECT * FROM users WHERE id=$1`, body.ReceiverID); err != nil {
		utils.Error(w, http.StatusBadRequest, "receiver not found")
		return
	}
	if !adjacentLayers(sender.Role, receiver.Role) {
		utils.Error(w, http.StatusForbidden, "queries are only allowed between adjacent layers")
		return
	}

	qID := uuid.New()
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO queries(id,project_id,sender_id,receiver_id,subject) VALUES($1,$2,$3,$4,$5)`,
		qID, projectID, senderID, body.ReceiverID, body.Subject)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "create failed")
		return
	}

	// First message
	_, _ = h.db.ExecContext(r.Context(),
		`INSERT INTO query_messages(id,query_id,sender_id,body) VALUES($1,$2,$3,$4)`,
		uuid.New(), qID, senderID, body.Message)

	h.hub.SendToUser(body.ReceiverID, string(models.NotifQueryReceived), map[string]interface{}{
		"query_id": qID, "project_id": projectID, "subject": body.Subject,
	})
	writeAudit(r.Context(), h.db, orgID, &senderID, "QUERY_CREATED", "QUERY", &qID, nil, nil)
	utils.Success(w, http.StatusCreated, map[string]interface{}{"id": qID})
}

// GET /api/v1/queries/{id}
func (h *QueryHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	userID := middleware.UserIDFrom(r.Context())

	var q models.Query
	if err := h.db.GetContext(r.Context(), &q,
		`SELECT * FROM queries WHERE id=$1 AND (sender_id=$2 OR receiver_id=$2)`, id, userID); err != nil {
		utils.Error(w, http.StatusNotFound, "query not found")
		return
	}

	var msgs []models.QueryMessage
	_ = h.db.SelectContext(r.Context(), &msgs,
		`SELECT * FROM query_messages WHERE query_id=$1 ORDER BY created_at`, id)
	q.Messages = msgs
	utils.Success(w, http.StatusOK, q)
}

// POST /api/v1/queries/{id}/messages
func (h *QueryHandler) PostMessage(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	senderID := middleware.UserIDFrom(r.Context())

	var q models.Query
	if err := h.db.GetContext(r.Context(), &q,
		`SELECT * FROM queries WHERE id=$1 AND (sender_id=$2 OR receiver_id=$2)`, id, senderID); err != nil {
		utils.Error(w, http.StatusNotFound, "query not found")
		return
	}
	if q.Status == models.QueryClosed {
		utils.Error(w, http.StatusBadRequest, "query is closed")
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := utils.ParseBody(r, &body); err != nil || body.Body == "" {
		utils.Error(w, http.StatusBadRequest, "body required")
		return
	}

	_, _ = h.db.ExecContext(r.Context(),
		`INSERT INTO query_messages(id,query_id,sender_id,body) VALUES($1,$2,$3,$4)`,
		uuid.New(), id, senderID, body.Body)

	// notify the other party
	recipientID := q.ReceiverID
	if senderID == q.ReceiverID {
		recipientID = q.SenderID
	}
	h.hub.SendToUser(recipientID, string(models.NotifQueryReceived), map[string]interface{}{
		"query_id": id, "project_id": q.ProjectID,
	})
	utils.SuccessMessage(w, http.StatusCreated, "message sent")
}

// PATCH /api/v1/queries/{id}/resolve
// Both sender and receiver must call this; query closes when both have.
func (h *QueryHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	userID := middleware.UserIDFrom(r.Context())

	var q models.Query
	if err := h.db.GetContext(r.Context(), &q,
		`SELECT * FROM queries WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "query not found")
		return
	}

	if userID == q.SenderID {
		_, _ = h.db.ExecContext(r.Context(),
			`UPDATE queries SET sender_resolved=true, updated_at=NOW() WHERE id=$1`, id)
	} else if userID == q.ReceiverID {
		_, _ = h.db.ExecContext(r.Context(),
			`UPDATE queries SET receiver_resolved=true, updated_at=NOW() WHERE id=$1`, id)
	} else {
		utils.Error(w, http.StatusForbidden, "not a participant of this query")
		return
	}

	// Reload and close if both resolved
	_ = h.db.GetContext(r.Context(), &q, `SELECT * FROM queries WHERE id=$1`, id)
	if q.SenderResolved && q.ReceiverResolved {
		_, _ = h.db.ExecContext(r.Context(),
			`UPDATE queries SET status='CLOSED', updated_at=NOW() WHERE id=$1`, id)
	}
	utils.SuccessMessage(w, http.StatusOK, "marked as resolved")
}

// adjacentLayers returns true if r1 and r2 are adjacent in the hierarchy.
func adjacentLayers(r1, r2 models.UserRole) bool {
	pairs := [][2]models.UserRole{
		{models.RoleAdmin, models.RoleLayer2},
		{models.RoleLayer2, models.RoleAdmin},
		{models.RoleLayer2, models.RoleLayer3},
		{models.RoleLayer3, models.RoleLayer2},
		{models.RoleSuperAdmin, models.RoleAdmin},
		{models.RoleAdmin, models.RoleSuperAdmin},
	}
	for _, p := range pairs {
		if p[0] == r1 && p[1] == r2 {
			return true
		}
	}
	return false
}
