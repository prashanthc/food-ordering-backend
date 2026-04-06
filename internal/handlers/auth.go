package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"food-ordering/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "email, name and password are required")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process registration")
		return
	}

	var user models.User
	err = h.db.QueryRow(
		`INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING id, email, name, created_at`,
		req.Email, req.Name, string(hash),
	).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := h.jwtService.GenerateToken(user.ID, user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusCreated, models.AuthResponse{Token: token, User: user})
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user models.User
	var passwordHash string
	err := h.db.QueryRow(
		`SELECT id, email, name, password_hash, created_at FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Email, &user.Name, &passwordHash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := h.jwtService.GenerateToken(user.ID, user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, models.AuthResponse{Token: token, User: user})
}
