package db

import (
	"database/sql"
	"log"
)

type seedProduct struct {
	Name     string
	Price    float64
	Category string
	ImageURL string
}

func Seed(db *sql.DB) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count); err != nil || count > 0 {
		return
	}

	products := []seedProduct{
		{
			Name:     "Belgian Waffle",
			Price:    12.99,
			Category: "Waffle",
			ImageURL: "https://images.unsplash.com/photo-1562376552-0d160a2f238d?w=400&h=300&fit=crop",
		},
		{
			Name:     "Chicken Waffle",
			Price:    15.99,
			Category: "Waffle",
			ImageURL: "https://images.unsplash.com/photo-1608039829572-78524f79c4c7?w=400&h=300&fit=crop",
		},
		{
			Name:     "Nutella Waffle",
			Price:    11.99,
			Category: "Waffle",
			ImageURL: "https://images.unsplash.com/photo-1567620905732-2d1ec7ab7445?w=400&h=300&fit=crop",
		},
		{
			Name:     "Classic Burger",
			Price:    13.99,
			Category: "Burger",
			ImageURL: "https://images.unsplash.com/photo-1568901346375-23c9450c58cd?w=400&h=300&fit=crop",
		},
		{
			Name:     "Cheese Burger",
			Price:    14.99,
			Category: "Burger",
			ImageURL: "https://images.unsplash.com/photo-1561758033-d89a9ad46330?w=400&h=300&fit=crop",
		},
		{
			Name:     "Double Patty Burger",
			Price:    18.99,
			Category: "Burger",
			ImageURL: "https://images.unsplash.com/photo-1553979459-d2229ba7433b?w=400&h=300&fit=crop",
		},
		{
			Name:     "Margherita Pizza",
			Price:    16.99,
			Category: "Pizza",
			ImageURL: "https://images.unsplash.com/photo-1574071318508-1cdbab80d002?w=400&h=300&fit=crop",
		},
		{
			Name:     "Pepperoni Pizza",
			Price:    19.99,
			Category: "Pizza",
			ImageURL: "https://images.unsplash.com/photo-1565299624946-b28f40a0ae38?w=400&h=300&fit=crop",
		},
		{
			Name:     "BBQ Chicken Pizza",
			Price:    21.99,
			Category: "Pizza",
			ImageURL: "https://images.unsplash.com/photo-1513104890138-7c749659a591?w=400&h=300&fit=crop",
		},
		{
			Name:     "Caesar Salad",
			Price:    9.99,
			Category: "Salad",
			ImageURL: "https://images.unsplash.com/photo-1550304943-4f24f54ddde9?w=400&h=300&fit=crop",
		},
		{
			Name:     "Greek Salad",
			Price:    10.99,
			Category: "Salad",
			ImageURL: "https://images.unsplash.com/photo-1540189549336-e6e99c3679fe?w=400&h=300&fit=crop",
		},
		{
			Name:     "Garden Fresh Salad",
			Price:    8.99,
			Category: "Salad",
			ImageURL: "https://images.unsplash.com/photo-1512621776951-a57141f2eefd?w=400&h=300&fit=crop",
		},
		{
			Name:     "Fresh Lemonade",
			Price:    4.99,
			Category: "Drink",
			ImageURL: "https://images.unsplash.com/photo-1621263764928-df1444c5e859?w=400&h=300&fit=crop",
		},
		{
			Name:     "Orange Juice",
			Price:    5.99,
			Category: "Drink",
			ImageURL: "https://images.unsplash.com/photo-1600271886742-f049cd451bba?w=400&h=300&fit=crop",
		},
		{
			Name:     "Cola",
			Price:    3.99,
			Category: "Drink",
			ImageURL: "https://images.unsplash.com/photo-1622483767028-3f66f32aef97?w=400&h=300&fit=crop",
		},
		{
			Name:     "Cold Brew Coffee",
			Price:    6.99,
			Category: "Drink",
			ImageURL: "https://images.unsplash.com/photo-1461023058943-07fcbe16d735?w=400&h=300&fit=crop",
		},
		{
			Name:     "Vanilla Ice Cream",
			Price:    7.99,
			Category: "Dessert",
			ImageURL: "https://images.unsplash.com/photo-1563805042-7684c019e1cb?w=400&h=300&fit=crop",
		},
		{
			Name:     "Chocolate Lava Cake",
			Price:    9.99,
			Category: "Dessert",
			ImageURL: "https://images.unsplash.com/photo-1563729784474-d77dbb933a9e?w=400&h=300&fit=crop",
		},
		{
			Name:     "Fudge Brownie",
			Price:    6.99,
			Category: "Dessert",
			ImageURL: "https://images.unsplash.com/photo-1515037893149-de7f840978e2?w=400&h=300&fit=crop",
		},
		{
			Name:     "Spicy Chicken Wrap",
			Price:    11.99,
			Category: "Wrap",
			ImageURL: "https://images.unsplash.com/photo-1626700051175-6818013e1d4f?w=400&h=300&fit=crop",
		},
		{
			Name:     "Veggie Wrap",
			Price:    10.99,
			Category: "Wrap",
			ImageURL: "https://images.unsplash.com/photo-1550507992-eb63ffee0847?w=400&h=300&fit=crop",
		},
		{
			Name:     "Fish Tacos",
			Price:    13.99,
			Category: "Tacos",
			ImageURL: "https://images.unsplash.com/photo-1565299585323-38d6b0865b47?w=400&h=300&fit=crop",
		},
		{
			Name:     "Street Tacos",
			Price:    12.99,
			Category: "Tacos",
			ImageURL: "https://images.unsplash.com/photo-1551504734-5ee1c4a1479b?w=400&h=300&fit=crop",
		},
	}

	stmt, err := db.Prepare(`INSERT INTO products (name, price, category, image_url) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		log.Printf("Failed to prepare seed statement: %v", err)
		return
	}
	defer stmt.Close()

	for _, p := range products {
		if _, err := stmt.Exec(p.Name, p.Price, p.Category, p.ImageURL); err != nil {
			log.Printf("Failed to seed product %s: %v", p.Name, err)
		}
	}

	log.Printf("Seeded %d products", len(products))
}
