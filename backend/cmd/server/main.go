package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
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
		AllowedOrigins:   corsOrigins(origin),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
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

func corsOrigins(raw string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 6)
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		out = append(out, v)
	}
	for _, p := range strings.Split(raw, ",") {
		add(p)
	}
	add("http://localhost:3000")
	add("http://127.0.0.1:3000")
	add("https://pvmoney.gowin.ir")
	return out
}
