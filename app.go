package main

import (
	"context"
	"database/sql"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"

	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"

	"github.com/jstolwijk/spotify-playlist-backup/migrations"
)

//go:embed templates/*
var resources embed.FS

var t = template.Must(template.ParseFS(resources, "templates/*"))

func getEnvOrFallback(name string, fallback string) string {
	val := os.Getenv(name)

	if val != "" {
		return val
	}

	return fallback
}

var (
	state   = uuid.New().String()
	port    = getEnvOrFallback("PORT", "8080")
	baseUrl = getEnvOrFallback("BASE_URL", "http://localhost:8080")
	auth    = spotifyauth.New(spotifyauth.WithRedirectURL(baseUrl+"/callback"), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopePlaylistReadCollaborative, spotifyauth.ScopePlaylistModifyPrivate, spotifyauth.ScopePlaylistReadPrivate))
)

var dsn = getEnvOrFallback("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")

func main() {
	ctx := context.Background()
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	migrator := migrate.NewMigrator(db, migrations.Migrations)
	migrator.Init(ctx)
	migrator.Migrate(ctx)

	setupHandlers()

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func setupHandlers() {
	http.HandleFunc("/", rootPath)

	http.HandleFunc("/login", loginPath)
	http.HandleFunc("/callback", completeAuthPath)

	http.HandleFunc("/home", authenticated(homePath))
	http.HandleFunc("/backup", authenticated(backupPath))
}
