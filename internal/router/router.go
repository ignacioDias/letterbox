package router

import (
	"letterbox/internal/database"
	"letterbox/internal/handlers"
	"net/http"
)

type Router interface {
	SetupRoutes() *http.ServeMux
}

type router struct {
	mux         *http.ServeMux
	authHandler handlers.AuthHandler
}

func NewRouter(db *database.Database) Router {
	return &router{
		mux:         http.NewServeMux(),
		authHandler: handlers.NewAuthHandler(db.AuthRepository),
	}
}

func (r *router) SetupRoutes() *http.ServeMux {

	r.mux.HandleFunc("POST /api/v1/auth/register", r.authHandler.RegisterUser)
	r.mux.HandleFunc("POST /api/v1/auth/login", r.authHandler.LoginUser)
	r.mux.HandleFunc("DELETE /api/v1/auth/logout", r.authHandler.LogoutUser)
	r.mux.HandleFunc("GET /api/v1/auth/me", r.authHandler.GetCurrentUser)

	return r.mux
}
