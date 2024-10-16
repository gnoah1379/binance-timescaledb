package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func OpenDB(cfg Config) *sqlx.DB {
	db, err := sqlx.Connect("pgx", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPwd,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName),
	)
	if err != nil {
		panic(err)
	}
	return db
}

type Repository struct {
	db *sqlx.DB
}

func (r *Repository) UpsertCandlestick(candlesticks []Candlestick) error {
	_, err := r.db.NamedExec(`INSERT INTO candlestick (time, symbol, open, high, low, close, volume)
VALUES (:time, :symbol, :open, :high, :low, :close, :volume)
ON CONFLICT (time, symbol) DO 
UPDATE SET open = EXCLUDED.open, high = EXCLUDED.high, low = EXCLUDED.low, close = EXCLUDED.close, volume = EXCLUDED.volume                        
`, candlesticks)
	return err
}

func (r *Repository) GetLastCandlestick(symbol string) (Candlestick, bool, error) {
	var candlestick Candlestick
	err := r.db.Get(&candlestick, "SELECT * FROM candlestick WHERE symbol = $1 ORDER BY time DESC LIMIT 1", symbol)
	if errors.Is(err, sql.ErrNoRows) {
		return candlestick, false, nil
	}
	return candlestick, true, err

}

func (r *Repository) GetEnabledSymbols() ([]Symbol, error) {
	var symbols []Symbol
	err := r.db.Select(&symbols, "SELECT symbol, name, enabled FROM symbol WHERE enabled = true")
	return symbols, err
}
