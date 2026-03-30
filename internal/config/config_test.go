package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()

	cfg := Load()

	if cfg.Addr != ":8080" {
		t.Errorf("expected :8080, got %s", cfg.Addr)
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("expected localhost, got %s", cfg.DBHost)
	}
	if cfg.DBPort != "3306" {
		t.Errorf("expected 3306, got %s", cfg.DBPort)
	}
	if cfg.DBName != "tcc" {
		t.Errorf("expected tcc, got %s", cfg.DBName)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("APP_ADDR", ":9090")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "root")
	os.Setenv("DB_PASS", "secret")
	os.Setenv("DB_NAME", "mydb")
	defer os.Clearenv()

	cfg := Load()

	if cfg.Addr != ":9090" {
		t.Errorf("expected :9090, got %s", cfg.Addr)
	}
	if cfg.DBHost != "db.example.com" {
		t.Errorf("expected db.example.com, got %s", cfg.DBHost)
	}
	if cfg.DBName != "mydb" {
		t.Errorf("expected mydb, got %s", cfg.DBName)
	}
}
