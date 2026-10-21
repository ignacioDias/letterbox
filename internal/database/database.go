package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Database struct {
	DB *sqlx.DB
}

func NewDatabase(db *sqlx.DB) *Database {
	return &Database{
		DB: db,
	}
}

func (db *Database) Init() error {
	tx, err := db.DB.Beginx()
	if err != nil {
		return fmt.Errorf("begin init db transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	for _, tableStmt := range SchemaStatements {
		if _, err := tx.Exec(tableStmt); err != nil {
			return fmt.Errorf("create table %s: %w", tableStmt, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit init db transaction: %w", err)
	}

	return nil
}

func (db *Database) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}
