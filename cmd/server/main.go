package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"food-ordering/internal/auth"
	"food-ordering/internal/cache"
	"food-ordering/internal/config"
	"food-ordering/internal/db"
	"food-ordering/internal/handlers"
	"food-ordering/internal/middleware"
	"food-ordering/internal/promo"
	"food-ordering/internal/resilience"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	resilience.Init()

	cfg := config.Load()

	database := db.Connect(cfg)
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	db.Seed(database)

	redisClient := cache.NewClient(cfg)

	promoValidator := promo.NewValidator(redisClient.RDB())
	if !promoValidator.IsReady(context.Background()) {
		slog.Warn("promo data not loaded — run the coupon-worker job first")
	}

	jwtService := auth.NewJWTService(cfg.JWTSecret)
	h := handlers.New(database, redisClient, jwtService, promoValidator)

	r := mux.NewRouter()
	r.Use(middleware.RequestLogger)

	r.HandleFunc("/health/live", h.Liveness).Methods("GET")
	r.HandleFunc("/health/ready", h.Readiness).Methods("GET")

	r.HandleFunc("/auth/register", h.Register).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/login", h.Login).Methods("POST", "OPTIONS")

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/product", h.ListProducts).Methods("GET", "OPTIONS")
	api.HandleFunc("/product/{productId:[0-9]+}", h.GetProduct).Methods("GET", "OPTIONS")
	api.Handle("/order", middleware.AuthRequired(jwtService)(http.HandlerFunc(h.PlaceOrder))).Methods("POST", "OPTIONS")
	api.Handle("/orders", middleware.AuthRequired(jwtService)(http.HandlerFunc(h.ListOrders))).Methods("GET", "OPTIONS")

	r.PathPrefix("/").Handler(spaHandler())

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000", "http://localhost"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "api_key", "Idempotency-Key"},
		AllowCredentials: true,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      c.Handler(r),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped gracefully")
}
