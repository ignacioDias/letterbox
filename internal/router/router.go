package router

import "net/http"

type Router interface {
	SetupRoutes() *http.ServeMux
}

type router struct {
	mux *http.ServeMux
}

func NewRouter() Router {
	return &router{
		mux: http.NewServeMux(),
	}
}

func (r *router) SetupRoutes() *http.ServeMux {
	return r.mux
}
