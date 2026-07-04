package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/oaknore/pms3/internal/middleware"
	"github.com/oaknore/pms3/internal/models"
	"github.com/oaknore/pms3/internal/services"
	"github.com/oaknore/pms3/internal/utils"
)

type FileHandler struct {
	db  *sqlx.DB
	s3  *services.S3Service
}

func NewFileHandler(db *sqlx.DB, s3 *services.S3Service) *FileHandler {
	return &FileHandler{db: db, s3: s3}
}

// POST /api/v1/files/upload
// Multipart form: file, owner_type, owner_id, project_id (optional)
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50 MB
		utils.Error(w, http.StatusBadRequest, "file too large or bad form")
		return
	}

	orgID := middleware.OrgIDFrom(r.Context())
	userID := middleware.UserIDFrom(r.Context())

	ownerType := models.FileOwnerType(r.FormValue("owner_type"))
	ownerIDStr := r.FormValue("owner_id")
	projectIDStr := r.FormValue("project_id")

	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid owner_id")
		return
	}

	fh, _, err := r.FormFile("file")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "file field required")
		return
	}
	defer fh.Close()

	// Re-open as FileHeader via ParseMultipartForm result
	_, fileHeaders, _ := r.FormFile("file")
	_ = fileHeaders
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		utils.Error(w, http.StatusBadRequest, "no file provided")
		return
	}
	fileHeader := files[0]

	prefix := "org-" + orgID.String() + "/" + string(ownerType) + "/" + ownerID.String()
	key, url, err := h.s3.UploadFile(r.Context(), prefix, fileHeader)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "upload failed: "+err.Error())
		return
	}

	id := uuid.New()
	var projectID *uuid.UUID
	if pid, err := uuid.Parse(projectIDStr); err == nil {
		projectID = &pid
	}

	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO file_assets(id,org_id,project_id,owner_type,owner_id,file_name,file_size,mime_type,s3_key,url,uploaded_by)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		id, orgID, projectID, ownerType, ownerID,
		fileHeader.Filename, fileHeader.Size,
		fileHeader.Header.Get("Content-Type"),
		key, url, userID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "db insert failed")
		return
	}

	utils.Success(w, http.StatusCreated, map[string]interface{}{
		"id":        id,
		"url":       url,
		"s3_key":    key,
		"file_name": fileHeader.Filename,
	})
}

// GET /api/v1/files?owner_type=X&owner_id=Y
func (h *FileHandler) ListByOwner(w http.ResponseWriter, r *http.Request) {
	ownerType := r.URL.Query().Get("owner_type")
	ownerIDStr := r.URL.Query().Get("owner_id")
	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid owner_id")
		return
	}
	var files []models.FileAsset
	_ = h.db.SelectContext(r.Context(), &files,
		`SELECT * FROM file_assets WHERE owner_type=$1 AND owner_id=$2 ORDER BY created_at DESC`,
		ownerType, ownerID)
	utils.Success(w, http.StatusOK, files)
}

// GET /api/v1/files/{id}/presign
func (h *FileHandler) Presign(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var f models.FileAsset
	if err := h.db.GetContext(r.Context(), &f, `SELECT * FROM file_assets WHERE id=$1`, id); err != nil {
		utils.Error(w, http.StatusNotFound, "file not found")
		return
	}
	signedURL, err := h.s3.PresignGet(r.Context(), f.S3Key)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "presign failed")
		return
	}
	utils.Success(w, http.StatusOK, map[string]string{"url": signedURL})
}

// DELETE /api/v1/files/{id}
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	orgID := middleware.OrgIDFrom(r.Context())
	var f models.FileAsset
	if err := h.db.GetContext(r.Context(), &f,
		`SELECT * FROM file_assets WHERE id=$1 AND org_id=$2`, id, orgID); err != nil {
		utils.Error(w, http.StatusNotFound, "file not found")
		return
	}
	_ = h.s3.DeleteFile(r.Context(), f.S3Key)
	_, _ = h.db.ExecContext(r.Context(), `DELETE FROM file_assets WHERE id=$1`, id)
	utils.SuccessMessage(w, http.StatusOK, "file deleted")
}
