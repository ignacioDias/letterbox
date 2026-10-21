package main

import (
	"letterbox/internal/router"
	"letterbox/internal/server"
)

func main() {
	router := router.NewRouter()
	server := server.NewServer("8080", router)
	server.StartServer(router)
}
