package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"krishblog/internal/analytics"
	"krishblog/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env when running locally
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	log.Println("connecting to postgres...")
	pg, err := database.NewPostgres(dbURL)
	if err != nil {
		log.Fatalf("postgres connect: %v", err)
	}
	defer pg.Close()

	log.Println("creating ent client...")
	client, err := database.NewEntClient(pg)
	if err != nil {
		log.Fatalf("ent client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Println("running migrations...")
	if err := database.RunMigrations(ctx, client); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("running custom migrations (comments, claps)...")
	if err := database.RunCustomMigrations(ctx, pg); err != nil {
		log.Fatalf("custom migration failed: %v", err)
	}

	log.Println("applying analytics schema patches (ip_hash, browser, os, recorded_at)...")
	analyticsRepo := analytics.NewRepository(pg)
	if err := analyticsRepo.ApplySchemaPatches(ctx); err != nil {
		log.Fatalf("analytics schema patch failed: %v", err)
	}

	fmt.Println("✓ migrations complete")
}
