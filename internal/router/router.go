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
	userHandler handlers.UserHandler
	auth        middleware.AuthenticationMiddleware
}

func NewRouter(db *database.Database) Router {
	return &router{
		mux:         http.NewServeMux(),
		authHandler: handlers.NewAuthHandler(db.UserRepository, db.SessionRepository),
		userHandler: handlers.NewUserHandler(db.UserRepository),
		auth:        middleware.NewAuthenticationMiddleware(db.SessionRepository),
	}
}

func (r *router) SetupRoutes() *http.ServeMux {

	r.mux.HandleFunc("POST /api/v1/auth/register", r.authHandler.RegisterUser)
	r.mux.HandleFunc("POST /api/v1/auth/login", r.authHandler.LoginUser)
	r.mux.HandleFunc("DELETE /api/v1/auth/logout", r.auth.Authenticate(r.authHandler.LogoutUser))

	r.mux.HandleFunc("GET /api/v1/users/me", r.auth.Authenticate(r.userHandler.GetCurrentUser))
	r.mux.HandleFunc("GET /api/v1/users/username/{username}", r.userHandler.GetUserByUsername)
	return r.mux
}
