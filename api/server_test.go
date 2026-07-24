package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestDecodedPathRoutingMiddleware(t *testing.T) {
	router := chi.NewRouter()
	router.Use(decodedPathRoutingMiddleware)

	var got string
	router.Route("/api/v1/suggestions", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Get("/archetype/{archetypeName}", func(res http.ResponseWriter, req *http.Request) {
				got = chi.URLParam(req, "archetypeName")
			})
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/archetype/Ancient%20Gear%20%26%20Friends", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got != "Ancient Gear & Friends" {
		t.Fatalf("expected decoded param, got %q", got)
	}
}
