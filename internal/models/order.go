package models

import "time"

type OrderItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type OrderRequest struct {
	CouponCode string      `json:"couponCode"`
	Items      []OrderItem `json:"items"`
}

type Order struct {
	ID          string      `json:"id"`
	Items       []OrderItem `json:"items"`
	Products    []Product   `json:"products"`
	TotalAmount float64     `json:"totalAmount"`
	Discount    float64     `json:"discount"`
	FinalAmount float64     `json:"finalAmount"`
	CouponCode  string      `json:"couponCode,omitempty"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"createdAt"`
}
