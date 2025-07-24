// Core package used to configure skc-deck-api api and its endpoints.
package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	"github.com/ygo-skc/skc-deck-api/db"
	cModel "github.com/ygo-skc/skc-go/common/model"
	cUtil "github.com/ygo-skc/skc-go/common/util"
)

const (
	apiContext = "/api/v1/deck"
	apiName    = "skc-deck-api"
)

var (
	skcDeckAPIDBInterface db.SKCDeckAPIDAO = db.SKCDeckAPIDAOImplementation{}
	serverAPIKey          string
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// verifies API Key from request header is the correct API Key
func verifyApiKey(headers http.Header) *cModel.APIError {
	clientKey := headers.Get("API-Key")

	if clientKey != serverAPIKey {
		slog.Error("Client is using incorrect API Key. Cannot process request.")
		return &cModel.APIError{Message: "Request has incorrect or missing API Key."}
	}

	return nil
}

// middleware used to verify API Key
func verifyAPIKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if err := verifyApiKey(req.Header); err != nil {
			res.Header().Add("Content-Type", "application/json")
			res.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(res).Encode(err)
		} else {
			next.ServeHTTP(res, req)
		}
	})
}

// sets common headers for response
func commonResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Content-Type", "application/json")
		res.Header().Add("Cache-Control", "max-age=300")

		// gzip
		if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			res.Header().Set("Content-Encoding", "gzip")
			zip := gzip.NewWriter(res)
			next.ServeHTTP(gzipResponseWriter{Writer: zip, ResponseWriter: res}, req)
			zip.Close()
		} else {
			next.ServeHTTP(res, req)
		}
	})
}

// Configures routes and their middle wares
// This method should be called before the environment is set up as the API Key will be set according to the value found in environment
func RunHttpServer() {
	serverAPIKey = cUtil.EnvMap["API_KEY"] // configure API Key
	router := chi.NewRouter()

	// common middleware
	router.Use(commonResponseMiddleware)

	router.Route(apiContext, func(r chi.Router) {
		// configure non-admin routes
		r.Group(func(r chi.Router) {
			r.Get("/status", getAPIStatusHandler)
			r.Post("/", submitNewDeckListHandler)
			r.Get("/card/{cardID:[0-9]{8}}", getDecksFeaturingCardHandler)
			r.Get("/{deckID:[0-9a-z]+}", getDeckListHandler)
		})

		// admin routes
		r.Group(func(r chi.Router) {
			r.Use(verifyAPIKeyMiddleware)
		})
	})

	// Cors
	corsOpts := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "https://dev.thesupremekingscastle.com", "https://thesupremekingscastle.com", "https://www.thesupremekingscastle.com"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodOptions,
		},

		AllowedHeaders: []string{
			"*", //or you can your header key values which you are using in your application
		},
	})

	serveTLS(router, corsOpts)
}

// Configures and starts an HTTPS server with TLS encryption.
// It combines the TLS certificate and CA bundle, and utilizes the private key.
// Finally, it applies CORS middleware.
func serveTLS(router *chi.Mux, corsOpts *cors.Cors) {
	cUtil.CombineCerts("certs")
	port := 9010
	slog.Info(fmt.Sprintf("API starting on port %d", port))

	if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", port), "certs/concatenated.crt", "certs/private.key", corsOpts.Handler(router)); err != nil {
		log.Fatalf("There was an error starting api server: %s", err)
	}
}
