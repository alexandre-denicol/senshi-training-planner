package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
	"github.com/alexandre/senshi-training-planner/backend/internal/categories"
	"github.com/alexandre/senshi-training-planner/backend/internal/config"
	"github.com/alexandre/senshi-training-planner/backend/internal/database"
	"github.com/alexandre/senshi-training-planner/backend/internal/professors"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	authStore := auth.NewPostgresStore(pool)
	authService := auth.NewService(authStore)
	authHandler := auth.NewHandler(authService, auth.NewLoginLimiter(), cfg.AppEnv)
	professorStore := professors.NewPostgresStore(pool)
	professorService := professors.NewService(professorStore)
	professorHandler := professors.NewHandler(professorService)
	categoryStore := categories.NewPostgresStore(pool)
	categoryService := categories.NewService(categoryStore)
	categoryHandler := categories.NewHandler(categoryService)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	authHandler.Register(mux)
	adminOnly := func(next http.Handler) http.Handler {
		return authHandler.Authenticate(auth.RequireAdmin(next))
	}
	mux.Handle("/professors", adminOnly(http.HandlerFunc(professorHandler.Collection)))
	mux.Handle("/professors/", adminOnly(http.HandlerFunc(professorHandler.Resource)))
	mux.Handle("/categories", adminOnly(http.HandlerFunc(categoryHandler.Collection)))
	mux.Handle("/categories/", adminOnly(http.HandlerFunc(categoryHandler.Resource)))

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("server listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
