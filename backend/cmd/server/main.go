package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"pvmoney/internal/api"
	"pvmoney/internal/db"
	"pvmoney/internal/notify"
	"pvmoney/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	dsn := env("DATABASE_URL", "postgres://pvmoney:pvmoney@localhost:5432/pvmoney?sslmode=disable")
	port := env("PORT", "8080")
	origin := env("CORS_ORIGIN", "http://localhost:3000")

	database, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	st := store.New(database)
	var tg api.TelegramSender
	if bot := notify.Start(context.Background(), st, os.Getenv("TELEGRAM_BOT_TOKEN")); bot != nil {
		tg = bot
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{origin, "http://localhost:3000", "http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
	}))
	r.Mount("/", api.New(st, tg))

	log.Printf("pvmoney api listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
