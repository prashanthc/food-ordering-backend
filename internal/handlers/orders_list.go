package handlers

import (
	"context"
	"net/http"
	"strconv"

	"food-ordering/internal/middleware"
	"food-ordering/internal/models"
)

func (h *Handlers) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dbQueryTimeout)
	defer cancel()

	rows, err := h.db.QueryContext(ctx,
		`SELECT id, COALESCE(coupon_code, ''), total_amount, discount, final_amount, status, created_at
		 FROM orders WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch orders")
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.CouponCode, &o.TotalAmount, &o.Discount, &o.FinalAmount, &o.Status, &o.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read orders")
			return
		}

		itemRows, err := h.db.QueryContext(ctx,
			`SELECT oi.product_id, oi.quantity, oi.price, p.name, p.category, COALESCE(p.image_url, '')
			 FROM order_items oi JOIN products p ON p.id = oi.product_id
			 WHERE oi.order_id = $1`, o.ID)
		if err == nil {
			for itemRows.Next() {
				var item models.OrderItem
				var p models.Product
				var pid int
				if err := itemRows.Scan(&pid, &item.Quantity, &p.Price, &p.Name, &p.Category, &p.ImageURL); err == nil {
					item.ProductID = strconv.Itoa(pid)
					p.ID = item.ProductID
					o.Items = append(o.Items, item)
					o.Products = append(o.Products, p)
				}
			}
			itemRows.Close()
		}

		orders = append(orders, o)
	}

	if orders == nil {
		orders = []models.Order{}
	}

	writeJSON(w, http.StatusOK, orders)
}

