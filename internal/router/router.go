package router

import (
	"letterbox/internal/database"
	"letterbox/internal/handlers"
	"letterbox/internal/middleware"
	"letterbox/internal/services"
	"net/http"
)

type Router interface {
	SetupRoutes() *http.ServeMux
}

type router struct {
	mux             *http.ServeMux
	authHandler     handlers.AuthHandler
	userHandler     handlers.UserHandler
	followHandler   handlers.FollowHandler
	blockHandler    handlers.BlockHandler
	reviewHandler   handlers.ReviewHandler
	reactionHandler handlers.ReactionHandler
	movieHandler    handlers.MovieHandler
	auth            middleware.AuthenticationMiddleware
}

func NewRouter(db *database.Database) Router {
	return &router{
		mux:             http.NewServeMux(),
		authHandler:     handlers.NewAuthHandler(db.UserRepository, db.SessionRepository),
		userHandler:     handlers.NewUserHandler(db.UserRepository, db.SessionRepository),
		followHandler:   handlers.NewFollowHandler(db.FollowRepository, db.UserRepository, db.BlockRepository),
		blockHandler:    handlers.NewBlockHandler(db.BlockRepository, db.UserRepository),
		reviewHandler:   handlers.NewReviewHandler(db.ReviewRepository, db.UserRepository, db.BlockRepository, db.MovieRepository),
		reactionHandler: handlers.NewReactionHandler(db.ReactionRepository),
		movieHandler:    handlers.NewMovieHandler(services.NewMovieService(db.MovieRepository)),
		auth:            middleware.NewAuthenticationMiddleware(db.SessionRepository),
	}
}

func (r *router) SetupRoutes() *http.ServeMux {

	// Auth routes
	r.mux.HandleFunc("POST /api/v1/auth/register", r.authHandler.RegisterUser)
	r.mux.HandleFunc("POST /api/v1/auth/login", r.authHandler.LoginUser)
	r.mux.HandleFunc("DELETE /api/v1/auth/logout", r.auth.Authenticate(r.authHandler.LogoutUser))

	// User routes
	r.mux.HandleFunc("GET /api/v1/users/me", r.auth.Authenticate(r.userHandler.GetCurrentUser))
	r.mux.HandleFunc("DELETE /api/v1/users/me", r.auth.Authenticate(r.userHandler.DeleteCurrentUser))
	r.mux.HandleFunc("PUT /api/v1/users/me", r.auth.Authenticate(r.userHandler.UpdateCurrentUser))
	r.mux.HandleFunc("PUT /api/v1/users/me/password", r.auth.Authenticate(r.userHandler.UpdateCurrentUserPassword))
	r.mux.HandleFunc("GET /api/v1/users/username/{username}", r.userHandler.GetUserByUsername)

	// Follow routes
	r.mux.HandleFunc("GET /api/v1/users/username/{username}/followers", r.followHandler.GetFollowersByUsername)
	r.mux.HandleFunc("GET /api/v1/users/username/{username}/followers/count", r.followHandler.GetFollowersCountByUsername)

	r.mux.HandleFunc("GET /api/v1/users/username/{username}/following", r.followHandler.GetFollowingByUsername)
	r.mux.HandleFunc("GET /api/v1/users/username/{username}/following/count", r.followHandler.GetFollowingCountByUsername)

	r.mux.HandleFunc("POST /api/v1/users/username/{username}/follow", r.auth.Authenticate(r.followHandler.FollowUserByUsername))
	r.mux.HandleFunc("DELETE /api/v1/users/username/{username}/unfollow", r.auth.Authenticate(r.followHandler.UnfollowUserByUsername))

	// Block routes
	r.mux.HandleFunc("POST /api/v1/users/username/{username}/block", r.auth.Authenticate(r.blockHandler.BlockUserByUsername))
	r.mux.HandleFunc("DELETE /api/v1/users/username/{username}/block", r.auth.Authenticate(r.blockHandler.UnblockUserByUsername))

	//Review routes
	r.mux.HandleFunc("GET /api/v1/users/username/{username}/reviews", r.auth.Authenticate(r.reviewHandler.GetReviewsFromUserByUsername))
	r.mux.HandleFunc("GET /api/v1/reviews/following", r.auth.Authenticate(r.reviewHandler.GetReviewsFromFollowingUsers))

	r.mux.HandleFunc("GET /api/v1/movies/{movie_id}/reviews", r.reviewHandler.GetReviewsFromMovieByID)
	r.mux.HandleFunc("GET /api/v1/movies/{movie_id}/reviews/average", r.reviewHandler.GetAverageRatingForMovieByID)
	r.mux.HandleFunc("GET /api/v1/movies/{movie_id}/reviews/count", r.reviewHandler.GetReviewsCountForMovieByID)
	r.mux.HandleFunc("POST /api/v1/movies/{movie_id}/reviews", r.auth.Authenticate(r.reviewHandler.CreateReviewForMovieByID))
	r.mux.HandleFunc("PATCH /api/v1/reviews/{review_id}", r.auth.Authenticate(r.reviewHandler.UpdateReviewByID))
	r.mux.HandleFunc("DELETE /api/v1/reviews/{review_id}", r.auth.Authenticate(r.reviewHandler.DeleteReviewByID))

	r.mux.HandleFunc("GET /api/v1/reviews/{review_id}", r.reviewHandler.GetReviewByID)

	// Like and Dislike routes
	r.mux.HandleFunc("POST /api/v1/reviews/{review_id}/like", r.auth.Authenticate(r.reactionHandler.LikeReviewByID))
	r.mux.HandleFunc("POST /api/v1/reviews/{review_id}/dislike", r.auth.Authenticate(r.reactionHandler.DislikeReviewByID))
	r.mux.HandleFunc("DELETE /api/v1/reviews/{review_id}/reaction", r.auth.Authenticate(r.reactionHandler.RemoveReactionFromReviewByID))

	r.mux.HandleFunc("POST /api/v1/comments/{comment_id}/like", r.auth.Authenticate(r.reactionHandler.LikeCommentByID))
	r.mux.HandleFunc("POST /api/v1/comments/{comment_id}/dislike", r.auth.Authenticate(r.reactionHandler.DislikeCommentByID))
	r.mux.HandleFunc("DELETE /api/v1/comments/{comment_id}/reaction", r.auth.Authenticate(r.reactionHandler.RemoveReactionFromCommentByID))

	// Movie routes
	r.mux.HandleFunc("GET /api/v1/movies/{movie_id}", r.movieHandler.GetMovieByID)
	r.mux.HandleFunc("GET /api/v1/movies/{movie_id}/similar", r.movieHandler.GetSimilarMovies)
	r.mux.HandleFunc("GET /api/v1/movies/{movie_id}/recommendations", r.movieHandler.GetRecommendationsForMovieByID)
	// r.mux.HandleFunc("GET /api/v1/movies/search", r.movieHandler.) //TODO
	r.mux.HandleFunc("GET /api/v1/movies/trending", r.movieHandler.GetTrendingMovies)

	// Comment routes

	// Messaging routes

	return r.mux
}
