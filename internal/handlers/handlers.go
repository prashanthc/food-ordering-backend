package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"food-ordering/internal/auth"
	"food-ordering/internal/cache"
	"food-ordering/internal/promo"
)

type Handlers struct {
	db             *sql.DB
	cache          *cache.Client
	jwtService     *auth.JWTService
	promoValidator *promo.Validator
}

func New(db *sql.DB, cache *cache.Client, jwtService *auth.JWTService, promoValidator *promo.Validator) *Handlers {
	return &Handlers{
		db:             db,
		cache:          cache,
		jwtService:     jwtService,
		promoValidator: promoValidator,
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
