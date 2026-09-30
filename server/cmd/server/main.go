// Command server runs the Kanban game API: migrations, demo seed, server-side timers and HTTP.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kanban-game/server/db"
	"kanban-game/server/internal/app"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	reset := flag.Bool("reset", false, "drop all data and recreate the demo database (local prototype only)")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := env("DATABASE_URL", "postgres://kanban:kanban-local-only@127.0.0.1:15433/kanban?sslmode=disable")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		log.Fatalf("database is not reachable (start it with: docker compose -f compose.local.yml up -d): %v", err)
	}
	if *reset {
		if _, err = pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
			log.Fatalf("reset: %v", err)
		}
		log.Print("database reset")
	}
	if err = db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	srv := app.New(pool, app.Config{
		Origins:   strings.Split(env("ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:8090,http://127.0.0.1:8090"), ","),
		StaticDir: env("STATIC_DIR", "../web/dist"),
	})
	seeded, err := srv.Seed(ctx)
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	if seeded {
		log.Print("demo data created")
	}
	go srv.RunTimers(ctx)

	addr := env("ADDR", "127.0.0.1:8090")
	hs := &http.Server{Addr: addr, Handler: srv.Routes(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = hs.Shutdown(shut)
	}()
	log.Printf("listening on http://%s (DEMO login without passwords — local prototype only)", addr)
	if err = hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
