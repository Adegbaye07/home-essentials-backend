package config_test

import (
	"testing"

	"homeessentials/backend/internal/config"
)

func TestLoad_DefaultCORS(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017/home_essentials")
	t.Setenv("CORS_ORIGINS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.CORSOrigins) != 2 {
		t.Fatalf("len(CORSOrigins) = %d, want 2", len(cfg.CORSOrigins))
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
}

func TestLoad_CustomCORS(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017/home_essentials")
	t.Setenv("CORS_ORIGINS", "http://a.test, http://b.test")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.CORSOrigins) != 2 {
		t.Fatalf("len(CORSOrigins) = %d, want 2", len(cfg.CORSOrigins))
	}
	if cfg.CORSOrigins[0] != "http://a.test" || cfg.CORSOrigins[1] != "http://b.test" {
		t.Fatalf("origins = %v", cfg.CORSOrigins)
	}
}
