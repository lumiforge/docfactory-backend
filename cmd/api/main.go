package main

import (
	"log"
	"net/http"
	"os"

	"github.com/lumiforge/docfactory-backend/internal/httpapi"
	"github.com/lumiforge/docfactory-backend/internal/templates"
)

func main() {
	repo := templates.NewInMemoryRepository()
	service := templates.NewTemplateService(repo)
	handler := httpapi.NewTemplateHandler(service)

	addr := ":8080"
	if v := os.Getenv("PORT"); v != "" {
		addr = ":" + v
	}

	log.Printf("starting API server on %s", addr)
	if err := http.ListenAndServe(addr, httpapi.Router(handler)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
