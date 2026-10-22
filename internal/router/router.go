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
	mux           *http.ServeMux
	authHandler   handlers.AuthHandler
	userHandler   handlers.UserHandler
	followHandler handlers.FollowHandler
	auth          middleware.AuthenticationMiddleware
}

func NewRouter(db *database.Database) Router {
	return &router{
		mux:           http.NewServeMux(),
		authHandler:   handlers.NewAuthHandler(db.UserRepository, db.SessionRepository),
		userHandler:   handlers.NewUserHandler(db.UserRepository),
		followHandler: handlers.NewFollowHandler(db.FollowRepository),
		auth:          middleware.NewAuthenticationMiddleware(db.SessionRepository),
	}
}

func (r *router) SetupRoutes() *http.ServeMux {

	r.mux.HandleFunc("POST /api/v1/auth/register", r.authHandler.RegisterUser)
	r.mux.HandleFunc("POST /api/v1/auth/login", r.authHandler.LoginUser)
	r.mux.HandleFunc("DELETE /api/v1/auth/logout", r.auth.Authenticate(r.authHandler.LogoutUser))

	r.mux.HandleFunc("GET /api/v1/users/me", r.auth.Authenticate(r.userHandler.GetCurrentUser))
	r.mux.HandleFunc("DELETE /api/v1/users/me", r.auth.Authenticate(r.userHandler.DeleteCurrentUser))
	r.mux.HandleFunc("PUT /api/v1/users/me", r.auth.Authenticate(r.userHandler.UpdateCurrentUser))
	r.mux.HandleFunc("GET /api/v1/users/username/{username}", r.userHandler.GetUserByUsername)

	r.mux.HandleFunc("GET /api/v1/users/username/{username}/followers", r.followHandler.GetFollowersByUsername)
	r.mux.HandleFunc("GET /api/v1/users/username/{username}/following", r.followHandler.GetFollowingByUsername)
	r.mux.HandleFunc("POST /api/v1/users/username/{username}/follow", r.followHandler.FollowUserByUsername)
	r.mux.HandleFunc("DELETE /api/v1/users/username/{username}/unfollow", r.followHandler.UnfollowUserByUsername)
	return r.mux
}
