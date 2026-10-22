package handlers

import (
	"letterbox/internal/middleware"
	"letterbox/internal/repositories"
	"letterbox/internal/services"
	"net/http"
	"os"
)

type AuthHandler interface {
	RegisterUser(w http.ResponseWriter, r *http.Request)
	LoginUser(w http.ResponseWriter, r *http.Request)
	LogoutUser(w http.ResponseWriter, r *http.Request)
}

type authHandler struct {
	authServices services.AuthService
	secureCookie bool
}

type RegisterUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Country  string `json:"country"`
}

type LoginUserRequest struct {
	EmailOrUsername string `json:"emailOrUsername"`
	Password        string `json:"password"`
}

var registerUserErrors = []errorMapping{
	{err: services.ErrInvalidUsername, status: http.StatusBadRequest},
	{err: services.ErrInvalidEmail, status: http.StatusBadRequest},
	{err: services.ErrInvalidPassword, status: http.StatusBadRequest},
	{err: services.ErrInvalidCountry, status: http.StatusBadRequest},
	{err: repositories.ErrUsernameAlreadyExists, status: http.StatusConflict},
	{err: repositories.ErrEmailAlreadyExists, status: http.StatusConflict},
}

var loginUserErrors = []errorMapping{
	{err: services.ErrInvalidCredentials, status: http.StatusUnauthorized, message: "Invalid credentials"},
	{err: repositories.ErrUserNotFound, status: http.StatusUnauthorized, message: "Invalid credentials"},
}

func NewAuthHandler(userRepo repositories.UserRepository, sessionRepo repositories.SessionRepository) AuthHandler {
	return &authHandler{
		authServices: services.NewAuthService(userRepo, sessionRepo),
		secureCookie: os.Getenv("APP_ENV") == "production",
	}
}

func (a *authHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var registerReq RegisterUserRequest
	if !decodeJSON(w, r, &registerReq) {
		return
	}
	user, err := a.authServices.RegisterUser(r.Context(), registerReq.Username, registerReq.Email, registerReq.Password, registerReq.Country)
	if err != nil {
		writeError(w, err, "register user", "Error creating the user", registerUserErrors...)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (a *authHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var loginReq LoginUserRequest
	if !decodeJSON(w, r, &loginReq) {
		return
	}
	user, err := a.authServices.LoginUser(r.Context(), loginReq.EmailOrUsername, loginReq.Password)
	if err != nil {
		writeError(w, err, "login user", "Error logging in the user", loginUserErrors...)
		return
	}
	session, err := a.authServices.CreateSession(r.Context(), user.ID)
	if err != nil {
		writeError(w, err, "login user", "Error creating the session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookie,
		SameSite: http.SameSiteStrictMode,
		Expires:  session.ExpiresAt,
	})
	writeJSON(w, http.StatusOK, user)
}

func (a *authHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := requireSessionID(w, r)
	if !ok {
		return
	}
	if err := a.authServices.DeleteSession(r.Context(), sessionID); err != nil {
		writeError(w, err, "logout user", "Error logging out the user")
		return
	}
	clearSessionCookie(w, a.secureCookie)
	w.WriteHeader(http.StatusOK)
}
