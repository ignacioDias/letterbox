package router

import (
	"letterbox/internal/database"
	"letterbox/internal/handlers"
	"letterbox/internal/middleware"
	"net/http"
)

type Router interface {
	SetupRoutes() *http.ServeMux
}

type router struct {
	mux         *http.ServeMux
	authHandler handlers.AuthHandler
	auth        middleware.AuthenticationMiddleware
}

func NewRouter(db *database.Database) Router {
	return &router{
		mux:         http.NewServeMux(),
		authHandler: handlers.NewAuthHandler(db.AuthRepository),
		auth:        middleware.NewAuthenticationMiddleware(db.AuthRepository),
	}
}

func (r *router) SetupRoutes() *http.ServeMux {

	r.mux.HandleFunc("POST /api/v1/auth/register", r.authHandler.RegisterUser)
	r.mux.HandleFunc("POST /api/v1/auth/login", r.authHandler.LoginUser)
	r.mux.HandleFunc("DELETE /api/v1/auth/logout", r.auth.Authenticate(r.authHandler.LogoutUser))
	r.mux.HandleFunc("GET /api/v1/auth/me", r.auth.Authenticate(r.authHandler.GetCurrentUser))

	return r.mux
}
