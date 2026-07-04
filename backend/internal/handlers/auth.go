package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/oaknore/pms3/internal/middleware"
	"github.com/oaknore/pms3/internal/models"
	"github.com/oaknore/pms3/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db            *sqlx.DB
	jwtSecret     string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewAuthHandler(db *sqlx.DB, jwtSecret string, accessExpiry, refreshExpiry time.Duration) *AuthHandler {
	return &AuthHandler{db: db, jwtSecret: jwtSecret, accessExpiry: accessExpiry, refreshExpiry: refreshExpiry}
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var user models.User
	err := h.db.GetContext(r.Context(), &user,
		`SELECT * FROM users WHERE email=$1 AND is_active=true`, body.Email)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		utils.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	accessToken, err := middleware.GenerateAccessToken(
		h.jwtSecret, h.accessExpiry, user.ID, user.OrgID, user.Role, user.DepartmentID,
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	refreshToken, tokenHash := generateRefreshToken()
	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO refresh_tokens(id,user_id,token_hash,expires_at)
		 VALUES($1,$2,$3,$4)`,
		uuid.New(), user.ID, tokenHash, time.Now().Add(h.refreshExpiry),
	)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "could not store refresh token")
		return
	}

	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE users SET last_login_at=$1 WHERE id=$2`, time.Now(), user.ID)

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(h.accessExpiry.Seconds()),
		"user": map[string]interface{}{
			"id":            user.ID,
			"email":         user.Email,
			"first_name":    user.FirstName,
			"last_name":     user.LastName,
			"role":          user.Role,
			"department_id": user.DepartmentID,
			"org_id":        user.OrgID,
		},
	})
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := utils.ParseBody(r, &body); err != nil || body.RefreshToken == "" {
		utils.Error(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	hash := hashToken(body.RefreshToken)
	var rt models.RefreshToken
	err := h.db.GetContext(r.Context(), &rt,
		`SELECT * FROM refresh_tokens WHERE token_hash=$1 AND revoked=false AND expires_at > NOW()`, hash)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	var user models.User
	if err = h.db.GetContext(r.Context(), &user, `SELECT * FROM users WHERE id=$1 AND is_active=true`, rt.UserID); err != nil {
		utils.Error(w, http.StatusUnauthorized, "user not found")
		return
	}

	// Rotate: revoke old, issue new
	_, _ = h.db.ExecContext(r.Context(), `UPDATE refresh_tokens SET revoked=true WHERE id=$1`, rt.ID)

	accessToken, _ := middleware.GenerateAccessToken(
		h.jwtSecret, h.accessExpiry, user.ID, user.OrgID, user.Role, user.DepartmentID,
	)
	newRefresh, newHash := generateRefreshToken()
	_, _ = h.db.ExecContext(r.Context(),
		`INSERT INTO refresh_tokens(id,user_id,token_hash,expires_at) VALUES($1,$2,$3,$4)`,
		uuid.New(), user.ID, newHash, time.Now().Add(h.refreshExpiry),
	)

	utils.Success(w, http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": newRefresh,
		"expires_in":    int(h.accessExpiry.Seconds()),
	})
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = utils.ParseBody(r, &body)
	if body.RefreshToken != "" {
		hash := hashToken(body.RefreshToken)
		_, _ = h.db.ExecContext(r.Context(), `UPDATE refresh_tokens SET revoked=true WHERE token_hash=$1`, hash)
	}
	utils.SuccessMessage(w, http.StatusOK, "logged out")
}

// GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r.Context())
	var user models.User
	if err := h.db.GetContext(r.Context(), &user, `SELECT * FROM users WHERE id=$1`, userID); err != nil {
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	}
	utils.Success(w, http.StatusOK, user)
}

// POST /api/v1/auth/change-password
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r.Context())
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := utils.ParseBody(r, &body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.NewPassword) < 8 {
		utils.Error(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	var user models.User
	if err := h.db.GetContext(r.Context(), &user, `SELECT * FROM users WHERE id=$1`, userID); err != nil {
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.CurrentPassword)); err != nil {
		utils.Error(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	_, _ = h.db.ExecContext(r.Context(), `UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2`, string(hash), userID)
	utils.SuccessMessage(w, http.StatusOK, "password updated")
}

// ── helpers ──────────────────────────────────────────────

func generateRefreshToken() (token, hash string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token = hex.EncodeToString(b)
	hash = hashToken(token)
	return
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
