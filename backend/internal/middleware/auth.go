package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/oaknore/pms3/internal/models"
	"github.com/oaknore/pms3/internal/utils"
)

type contextKey string

const (
	ContextKeyUserID   contextKey = "user_id"
	ContextKeyOrgID    contextKey = "org_id"
	ContextKeyRole     contextKey = "role"
	ContextKeyDeptID   contextKey = "dept_id"
)

type Claims struct {
	UserID uuid.UUID        `json:"user_id"`
	OrgID  uuid.UUID        `json:"org_id"`
	Role   models.UserRole  `json:"role"`
	DeptID *uuid.UUID       `json:"dept_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed JWT access token.
func GenerateAccessToken(secret string, expiry time.Duration, userID, orgID uuid.UUID, role models.UserRole, deptID *uuid.UUID) (string, error) {
	claims := Claims{
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
		DeptID: deptID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "pms3",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// parseToken validates and parses a JWT string.
func parseToken(tokenStr, secret string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// Authenticate is the JWT verification middleware. It reads Bearer token from
// Authorization header, validates it, and injects claims into context.
func Authenticate(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				utils.Error(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := parseToken(tokenStr, secret)
			if err != nil {
				utils.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyOrgID, claims.OrgID)
			ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
			ctx = context.WithValue(ctx, ContextKeyDeptID, claims.DeptID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoles enforces that the authenticated user has one of the given roles.
func RequireRoles(roles ...models.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[models.UserRole]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(ContextKeyRole).(models.UserRole)
			if _, ok := allowed[role]; !ok {
				utils.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ── Context helpers ──────────────────────────────────────

func UserIDFrom(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ContextKeyUserID).(uuid.UUID)
	return v
}

func OrgIDFrom(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ContextKeyOrgID).(uuid.UUID)
	return v
}

func RoleFrom(ctx context.Context) models.UserRole {
	v, _ := ctx.Value(ContextKeyRole).(models.UserRole)
	return v
}

func DeptIDFrom(ctx context.Context) *uuid.UUID {
	v, _ := ctx.Value(ContextKeyDeptID).(*uuid.UUID)
	return v
}
