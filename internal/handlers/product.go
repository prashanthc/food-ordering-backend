package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"food-ordering/internal/models"

	"github.com/gorilla/mux"
)

func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("products:cat:%s:search:%s", category, search)

	ctx := context.Background()
	var products []models.Product
	if err := h.cache.Get(ctx, cacheKey, &products); err == nil {
		writeJSON(w, http.StatusOK, products)
		return
	}

	query := `SELECT id, name, price, category, COALESCE(image_url, '') FROM products WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if category != "" && strings.ToLower(category) != "all" {
		query += fmt.Sprintf(" AND LOWER(category) = LOWER($%d)", argIdx)
		args = append(args, category)
		argIdx++
	}
	if search != "" {
		query += fmt.Sprintf(" AND LOWER(name) LIKE LOWER($%d)", argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	query += " ORDER BY category, name"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch products")
		return
	}
	defer rows.Close()

	products = []models.Product{}
	for rows.Next() {
		var p models.Product
		var id int
		if err := rows.Scan(&id, &p.Name, &p.Price, &p.Category, &p.ImageURL); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read products")
			return
		}
		p.ID = strconv.Itoa(id)
		products = append(products, p)
	}

	h.cache.Set(ctx, cacheKey, products, 5*time.Minute)

	writeJSON(w, http.StatusOK, products)
}

func (h *Handlers) GetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productIDStr := vars["productId"]

	productID, err := strconv.Atoi(productIDStr)
	if err != nil || productID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	cacheKey := fmt.Sprintf("product:%d", productID)
	ctx := context.Background()

	var product models.Product
	if err := h.cache.Get(ctx, cacheKey, &product); err == nil {
		writeJSON(w, http.StatusOK, product)
		return
	}

	var id int
	err = h.db.QueryRow(
		`SELECT id, name, price, category, COALESCE(image_url, '') FROM products WHERE id = $1`,
		productID,
	).Scan(&id, &product.Name, &product.Price, &product.Category, &product.ImageURL)
	if err != nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	product.ID = strconv.Itoa(id)

	h.cache.Set(ctx, cacheKey, product, 10*time.Minute)

	writeJSON(w, http.StatusOK, product)
}
