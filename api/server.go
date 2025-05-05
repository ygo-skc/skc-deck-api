// Core package used to configure skc-deck-api api and its endpoints.
package api

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/ygo-skc/skc-deck-api/db"
	cModel "github.com/ygo-skc/skc-go/common/model"
	cUtil "github.com/ygo-skc/skc-go/common/util"
)

const (
	CONTEXT = "/api/v1/deck"
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
			defer zip.Close()
			next.ServeHTTP(gzipResponseWriter{Writer: zip, ResponseWriter: res}, req)
		} else {
			next.ServeHTTP(res, req)
		}
	})
}

// Configures routes and their middle wares
// This method should be called before the environment is set up as the API Key will be set according to the value found in environment
func RunHttpServer() {
	serverAPIKey = cUtil.EnvMap["API_KEY"] // configure API Key
	router := mux.NewRouter()

	// configure non-admin routes
	unprotectedRoutes := router.PathPrefix(CONTEXT).Subrouter()
	unprotectedRoutes.HandleFunc("/status", getAPIStatusHandler)
	unprotectedRoutes.HandleFunc("", submitNewDeckListHandler).Methods(http.MethodPost).Name("Deck List Submission")
	unprotectedRoutes.HandleFunc("/card/{cardID:[0-9]{8}}", getDecksFeaturingCardHandler).Methods(http.MethodGet).Name("Deck Featuring Card")
	unprotectedRoutes.HandleFunc("/{deckID:[0-9a-z]+}", getDeckListHandler).Methods(http.MethodGet).Name("Retrieve Info On Deck")

	// admin routes
	protectedRoutes := router.PathPrefix(CONTEXT).Subrouter()
	protectedRoutes.Use(verifyAPIKeyMiddleware)

	// common middleware
	router.Use(commonResponseMiddleware)

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

// configure server to handle HTTPS (secured) calls
func serveTLS(router *mux.Router, corsOpts *cors.Cors) {
	slog.Debug("Starting server in port 9010 (secured)")

	cUtil.CombineCerts("certs")
	if err := http.ListenAndServeTLS(":9010", "certs/concatenated.crt", "certs/private.key", corsOpts.Handler(router)); err != nil {
		log.Fatalf("There was an error starting api server: %s", err)
	}
}
