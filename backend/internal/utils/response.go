package utils

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

type PaginatedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func Success(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, Response{Success: true, Data: data})
}

func SuccessMessage(w http.ResponseWriter, status int, message string) {
	JSON(w, status, Response{Success: true, Message: message})
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, Response{Success: false, Error: message})
}

func Paginated(w http.ResponseWriter, data interface{}, meta PaginationMeta) {
	JSON(w, http.StatusOK, PaginatedResponse{Success: true, Data: data, Meta: meta})
}

func ParseBody(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func PaginationParams(r *http.Request) (page, limit, offset int) {
	page = 1
	limit = 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v := atoi(p); v > 0 {
			page = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v := atoi(l); v > 0 && v <= 100 {
			limit = v
		}
	}
	offset = (page - 1) * limit
	return
}

func atoi(s string) int {
	v := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		v = v*10 + int(c-'0')
	}
	return v
}
