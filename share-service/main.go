// Share service for Qsim.
//
// Serves the compiled web app (static files) plus an API to publish circuits
// as short links:
//
//	POST /api/share        body: raw circuit text  -> 201 {"id":"..."}
//	GET  /api/share/<id>   -> 200 {"id":"...","data":"..."}
//	GET  /healthz          -> 200 ok
//
// Everything else is served from STATIC_DIR (default "./static").
//
// Storage is selected with STORAGE=disk (default; files under
// QSIM_SHARES_DIR, default "./data") or STORAGE=s3 (see storage_s3.go).
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const maxShareSize = 2 << 20 // 2 MiB of circuit text is plenty

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	backend := os.Getenv("STORAGE")
	if backend == "" {
		backend = "disk"
	}

	st, err := makeStore(backend)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	if err := openStore(st); err != nil {
		log.Fatalf("storage open: %v", err)
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}

	mux := http.NewServeMux()
	mux.Handle("/api/share", withCORS(shareHandler(st)))
	mux.Handle("/api/share/", withCORS(shareHandler(st)))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.Handle("/", withCORS(http.FileServer(http.Dir(staticDir))))

	addr := ":" + port
	log.Printf("share-service: http://localhost%s (storage=%s, static=%s)", addr, backend, staticDir)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func shareHandler(st store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/share/")

		switch r.Method {
		case http.MethodGet:
			if id == "" || strings.Contains(id, "/") {
				http.Error(w, "missing share id", http.StatusBadRequest)
				return
			}
			data, err := st.load(id)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"id": id, "data": data})
		case http.MethodPost:
			body, err := io.ReadAll(io.LimitReader(r.Body, maxShareSize))
			if err != nil {
				http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(string(body)) == "" {
				http.Error(w, "empty body", http.StatusBadRequest)
				return
			}
			id := randomID(10)
			if err := st.save(id, string(body)); err != nil {
				http.Error(w, "store: "+err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]string{"id": id})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// withCORS allows cross-origin requests so a separately hosted frontend can
// call the API. Shared circuits are public by design.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func randomID(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("q%x", time.Now().UnixNano())[:n]
	}
	return base64.RawURLEncoding.EncodeToString(b)[:n]
}