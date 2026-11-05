package database

import (
	"embed"
	"fmt"
	"letterbox/internal/repositories"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

type Database struct {
	DB                     *sqlx.DB
	UserRepository         repositories.UserRepository
	SessionRepository      repositories.SessionRepository
	FollowRepository       repositories.FollowRepository
	BlockRepository        repositories.BlockRepository
	MovieRepository        repositories.MovieRepository
	ReviewRepository       repositories.ReviewRepository
	ReactionRepository     repositories.ReactionRepository
	UserIdentityRepository repositories.UserIdentityRepository
}

func NewDatabase(db *sqlx.DB) *Database {
	return &Database{
		DB:                     db,
		UserRepository:         repositories.NewUserRepository(db),
		SessionRepository:      repositories.NewSessionRepository(db),
		FollowRepository:       repositories.NewFollowRepository(db),
		BlockRepository:        repositories.NewBlockRepository(db),
		MovieRepository:        repositories.NewMovieRepository(db),
		ReviewRepository:       repositories.NewReviewRepository(db),
		ReactionRepository:     repositories.NewReactionRepository(db),
		UserIdentityRepository: repositories.NewUserIdentityRepository(db),
	}
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (db *Database) Init() error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}

	if err := goose.Up(db.DB.DB, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func (db *Database) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}
