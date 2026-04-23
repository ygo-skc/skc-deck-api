// Core package used to configure skc-deck-api api and its endpoints.
package api

import (
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	"github.com/ygo-skc/skc-deck-api/db"
	cModel "github.com/ygo-skc/skc-go/common/v2/model"
	cUtil "github.com/ygo-skc/skc-go/common/v2/util"
	"golang.org/x/net/http2"
)

const (
	apiContext = "/api/v1/deck"
	apiName    = "skc-deck-api"
)

var (
	skcDeckAPIDBInterface db.SKCDeckAPIDAO = db.SKCDeckAPIDAOImplementation{}
	serverAPIKey          string

	gzipPool = sync.Pool{
		New: func() any {
			w, _ := gzip.NewWriterLevel(io.Discard, 2)
			return w
		},
	}
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
		if acceptsGzip(req) {
			zip := gzipPool.Get().(*gzip.Writer)
			zip.Reset(res)
			defer zip.Close()
			defer gzipPool.Put(zip)

			res.Header().Set("Content-Encoding", "gzip")
			res.Header().Del("Content-Length")
			next.ServeHTTP(gzipResponseWriter{Writer: zip, ResponseWriter: res}, req)
		} else {
			next.ServeHTTP(res, req)
		}
	})
}

func acceptsGzip(req *http.Request) bool {
	for val := range strings.SplitSeq(req.Header.Get("Accept-Encoding"), ",") {
		if strings.TrimSpace(strings.Split(val, ";")[0]) == "gzip" {
			return true
		}
	}
	return false
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

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS13,
		NextProtos: []string{"h2"},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
	}

	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", port),
		Handler:   corsOpts.Handler(router),
		TLSConfig: tlsCfg,

		ReadTimeout:       6 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      4 * time.Second,
		IdleTimeout:       15 * time.Second,

		MaxHeaderBytes: 64 << 10,
	}

	if err := http2.ConfigureServer(server, &http2.Server{
		MaxConcurrentStreams:         100,
		MaxHandlers:                  25,
		IdleTimeout:                  15 * time.Second,
		WriteByteTimeout:             4 * time.Second,
		MaxUploadBufferPerConnection: 10 << 10,
		MaxUploadBufferPerStream:     10 << 10,
	}); err != nil {
		log.Fatalf("Failed to configure HTTP/2: %v", err)
	}

	slog.Info(fmt.Sprintf("API starting on port %d", port))

	if err := server.ListenAndServeTLS("certs/concatenated.crt", "certs/private.key"); err != nil {
		log.Fatalf("There was an error starting api server: %s", err)
	}
}
