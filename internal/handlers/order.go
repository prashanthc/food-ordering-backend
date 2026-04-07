package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"food-ordering/internal/middleware"
	"food-ordering/internal/models"
)

const couponDiscountPercent = 0.20

func (h *Handlers) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	idempotencyKey := r.Header.Get("Idempotency-Key")

	if idempotencyKey != "" {
		var existing models.Order
		err := h.db.QueryRowContext(ctx,
			`SELECT id, coupon_code, total_amount, discount, final_amount, status, created_at
			 FROM orders WHERE user_id = $1 AND idempotency_key = $2`,
			userID, idempotencyKey,
		).Scan(&existing.ID, &sql.NullString{}, &existing.TotalAmount,
			&existing.Discount, &existing.FinalAmount, &existing.Status, &existing.CreatedAt)
		if err == nil {
			writeJSON(w, http.StatusOK, existing)
			return
		}
	}

	var req models.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Items) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "order must contain at least one item")
		return
	}

	for _, item := range req.Items {
		if item.ProductID == "" {
			writeError(w, http.StatusUnprocessableEntity, "productId is required for each item")
			return
		}
		if item.Quantity <= 0 {
			writeError(w, http.StatusUnprocessableEntity, "quantity must be greater than 0")
			return
		}
	}

	if req.CouponCode != "" {
		if !h.promoValidator.IsValid(ctx, req.CouponCode) {
			writeError(w, http.StatusUnprocessableEntity, "invalid coupon code")
			return
		}
		var priorUses int
		h.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM orders WHERE user_id = $1 AND coupon_code = $2`,
			userID, req.CouponCode,
		).Scan(&priorUses)
		if priorUses > 0 {
			writeError(w, http.StatusUnprocessableEntity, "coupon already used")
			return
		}
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	products := make([]models.Product, 0, len(req.Items))
	var totalAmount float64

	for _, item := range req.Items {
		productID, err := strconv.Atoi(item.ProductID)
		if err != nil || productID <= 0 {
			writeError(w, http.StatusBadRequest, "invalid productId: "+item.ProductID)
			return
		}

		var p models.Product
		var id int
		err = tx.QueryRowContext(ctx,
			`SELECT id, name, price, category, COALESCE(image_url, '') FROM products WHERE id = $1`,
			productID,
		).Scan(&id, &p.Name, &p.Price, &p.Category, &p.ImageURL)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "product not found: "+item.ProductID)
			return
		}
		p.ID = strconv.Itoa(id)
		products = append(products, p)
		totalAmount += p.Price * float64(item.Quantity)
	}

	totalAmount = math.Round(totalAmount*100) / 100

	discount := 0.0
	if req.CouponCode != "" {
		discount = math.Round(totalAmount*couponDiscountPercent*100) / 100
	}
	finalAmount := math.Round((totalAmount-discount)*100) / 100

	couponArg := interface{}(nil)
	if req.CouponCode != "" {
		couponArg = req.CouponCode
	}

	idempArg := interface{}(nil)
	if idempotencyKey != "" {
		idempArg = idempotencyKey
	}

	var order models.Order
	err = tx.QueryRowContext(ctx,
		`INSERT INTO orders (user_id, idempotency_key, coupon_code, total_amount, discount, final_amount)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, status, created_at`,
		userID, idempArg, couponArg, totalAmount, discount, finalAmount,
	).Scan(&order.ID, &order.Status, &order.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	itemStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO order_items (order_id, product_id, quantity, price) VALUES ($1, $2, $3, $4)`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to prepare order items")
		return
	}
	defer itemStmt.Close()

	for i, item := range req.Items {
		productID, _ := strconv.Atoi(item.ProductID)
		if _, err := itemStmt.ExecContext(ctx, order.ID, productID, item.Quantity, products[i].Price); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save order items")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit order")
		return
	}

	order.Items = req.Items
	order.Products = products
	order.TotalAmount = totalAmount
	order.Discount = discount
	order.FinalAmount = finalAmount
	if req.CouponCode != "" {
		order.CouponCode = req.CouponCode
	}

	writeJSON(w, http.StatusOK, order)
}
