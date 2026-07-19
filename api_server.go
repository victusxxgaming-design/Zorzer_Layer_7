package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// StartAPIServer registers all routes, applies middleware, and blocks on Listen.
// Port defaults to 8080 if port == 0.
func StartAPIServer(port int) {
	if port == 0 {
		port = 8080
	}

	mux := http.NewServeMux()

	// ── routes ────────────────────────────────────────────────────────────────
	mux.HandleFunc("/api/attack/start",  onlyMethod(http.MethodPost, handleStart))
	mux.HandleFunc("/api/attack/stop",   onlyMethod(http.MethodPost, handleStop))
	mux.HandleFunc("/api/attack/status", onlyMethod(http.MethodGet,  handleStatus))
	mux.HandleFunc("/api/health",        onlyMethod(http.MethodGet,  handleHealth))

	// ── server ────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      chain(mux, middlewareCORS, middlewareLogger),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	printAPISplash(port)
	log.Fatal(srv.ListenAndServe())
}

// ── middleware ────────────────────────────────────────────────────────────────

type middleware func(http.Handler) http.Handler

// chain wraps handler with each middleware (last middleware = outermost layer).
func chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// middlewareCORS adds permissive CORS headers and handles preflight requests.
func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// middlewareLogger logs every incoming request with method, path, and duration.
func middlewareLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("[api] %s %s → %d (%s)", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// onlyMethod returns 405 if the request method doesn't match.
func onlyMethod(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			fail(w, http.StatusMethodNotAllowed, "method not allowed; expected "+method)
			return
		}
		h(w, r)
	}
}

// ── splash ────────────────────────────────────────────────────────────────────

func printAPISplash(port int) {
	const (
		reset  = "\033[0m"
		cyan   = "\033[36m"
		green  = "\033[32m"
		yellow = "\033[33m"
		bold   = "\033[1m"
		dim    = "\033[2m"
	)
	fmt.Println(bold + cyan + `
  ███████╗██╗      █████╗ ██╗   ██╗███████╗██████╗      █████╗ ██████╗ ██╗
  ██╔════╝██║     ██╔══██╗╚██╗ ██╔╝██╔════╝██╔══██╗    ██╔══██╗██╔══██╗██║
  ███████╗██║     ███████║ ╚████╔╝ █████╗  ██████╔╝    ███████║██████╔╝██║
  ╚════██║██║     ██╔══██║  ╚██╔╝  ██╔══╝  ██╔══██╗    ██╔══██║██╔═══╝ ██║
  ███████║███████╗██║  ██║   ██║   ███████╗██║  ██║    ██║  ██║██║     ██║
  ╚══════╝╚══════╝╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝    ╚═╝  ╚═╝╚═╝     ╚═╝` + reset)
	fmt.Println()
	fmt.Println(dim + "  ┌──────────────────────────────────────────────────┐" + reset)
	fmt.Println(dim + "  │" + bold + cyan + "               API SERVER READY                   " + reset + dim + "│" + reset)
	fmt.Println(dim + "  ├──────────────────────────────────────────────────┤" + reset)
	fmt.Printf(dim+"  │"+reset+"  "+green+"POST"+reset+"  %-44s"+dim+"│"+reset+"\n", fmt.Sprintf("http://0.0.0.0:%d/api/attack/start", port))
	fmt.Printf(dim+"  │"+reset+"  "+yellow+"POST"+reset+"  %-44s"+dim+"│"+reset+"\n", fmt.Sprintf("http://0.0.0.0:%d/api/attack/stop", port))
	fmt.Printf(dim+"  │"+reset+"  "+cyan+"GET"+reset+"   %-44s"+dim+"│"+reset+"\n", fmt.Sprintf("http://0.0.0.0:%d/api/attack/status", port))
	fmt.Printf(dim+"  │"+reset+"  "+cyan+"GET"+reset+"   %-44s"+dim+"│"+reset+"\n", fmt.Sprintf("http://0.0.0.0:%d/api/health", port))
	fmt.Println(dim + "  └──────────────────────────────────────────────────┘" + reset)
	fmt.Println()
}
