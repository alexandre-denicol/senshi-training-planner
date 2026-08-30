package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
	"github.com/alexandre/senshi-training-planner/backend/internal/blocks"
	"github.com/alexandre/senshi-training-planner/backend/internal/categories"
	"github.com/alexandre/senshi-training-planner/backend/internal/config"
	"github.com/alexandre/senshi-training-planner/backend/internal/database"
	"github.com/alexandre/senshi-training-planner/backend/internal/history"
	"github.com/alexandre/senshi-training-planner/backend/internal/professors"
	"github.com/alexandre/senshi-training-planner/backend/internal/schedule"
	"github.com/alexandre/senshi-training-planner/backend/internal/students"
	"github.com/alexandre/senshi-training-planner/backend/internal/workouts"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL, string(cfg.AppEnv))
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
	blockStore := blocks.NewPostgresStore(pool)
	blockService := blocks.NewService(blockStore)
	blockHandler := blocks.NewHandler(blockService)
	studentStore := students.NewPostgresStore(pool)
	studentService := students.NewService(studentStore)
	studentHandler := students.NewHandler(studentService)
	workoutStore := workouts.NewPostgresStore(pool)
	workoutService := workouts.NewService(workoutStore)
	workoutHandler := workouts.NewHandler(workoutService)
	scheduleStore := schedule.NewPostgresStore(pool)
	scheduleService := schedule.NewService(scheduleStore)
	scheduleHandler := schedule.NewHandler(scheduleService)
	historyStore := history.NewPostgresStore(pool)
	historyService := history.NewService(historyStore)
	historyHandler := history.NewHandler(historyService)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	authHandler.Register(mux)
	authenticated := func(next http.Handler) http.Handler {
		return authHandler.Authenticate(next)
	}
	adminOnly := func(next http.Handler) http.Handler {
		return authHandler.Authenticate(auth.RequireAdmin(next))
	}
	mux.Handle("/professors", adminOnly(http.HandlerFunc(professorHandler.Collection)))
	mux.Handle("/professors/", adminOnly(http.HandlerFunc(professorHandler.Resource)))
	mux.Handle("/categories", authenticated(http.HandlerFunc(categoryHandler.Collection)))
	mux.Handle("/categories/", authenticated(http.HandlerFunc(categoryHandler.Resource)))
	mux.Handle("/blocks", authenticated(http.HandlerFunc(blockHandler.Collection)))
	mux.Handle("/blocks/", authenticated(http.HandlerFunc(blockHandler.Resource)))
	mux.Handle("/students", authenticated(http.HandlerFunc(studentHandler.Collection)))
	mux.Handle("/students/", authenticated(http.HandlerFunc(studentHandler.Resource)))
	mux.Handle("/workouts", authenticated(http.HandlerFunc(workoutHandler.Collection)))
	mux.Handle("/workouts/", authenticated(http.HandlerFunc(workoutHandler.Resource)))
	mux.Handle("/schedule", authenticated(http.HandlerFunc(scheduleHandler.Collection)))
	mux.Handle("/schedule/", authenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if scheduleCompletePath(r.URL.Path) {
			historyHandler.CompleteSchedule(w, r, scheduleCompleteID(r.URL.Path))
			return
		}
		scheduleHandler.Resource(w, r)
	})))
	mux.Handle("/history", authenticated(http.HandlerFunc(historyHandler.Collection)))
	mux.Handle("/history/", authenticated(http.HandlerFunc(historyHandler.Resource)))

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

func scheduleCompletePath(path string) bool {
	segments := strings.Split(strings.Trim(strings.TrimPrefix(path, "/schedule/"), "/"), "/")
	return len(segments) == 2 && segments[0] != "" && segments[1] == "complete"
}

func scheduleCompleteID(path string) string {
	segments := strings.Split(strings.Trim(strings.TrimPrefix(path, "/schedule/"), "/"), "/")
	if len(segments) == 0 {
		return ""
	}

	return segments[0]
}
