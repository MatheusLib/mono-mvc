package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func NewMySQL(ctx context.Context, host, port, user, pass, db string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci", user, pass, host, port, db)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return configure(ctx, conn)
}

func configure(ctx context.Context, conn *sql.DB) (*sql.DB, error) {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := conn.PingContext(pingCtx); err != nil {
		_ = conn.Close()
		return nil, err
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	return conn, nil
}
