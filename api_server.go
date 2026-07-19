package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// killPort forcefully frees the given TCP port before binding to it.
func killPort(port int) {
	cmd := exec.Command("fuser", "-k", fmt.Sprintf("%d/tcp", port))
	_ = cmd.Run() // ignore error — port may already be free
}

// StartAPIServer registers all routes, applies middleware, and blocks on Listen.
// Port defaults to 8080 if port == 0.
func StartAPIServer(port int) {
	if port == 0 {
		port = 8080
	}

	// ── free the port ─────────────────────────────────────────────────────────
	killPort(port)

	// ── supabase self-registration ────────────────────────────────────────────
	sb, err := newSupabaseClient()
	if err != nil {
		log.Printf("[supabase] skipping registration: %v", err)
	} else {
		if err := RegisterSelf(sb); err != nil {
			log.Printf("[supabase] registration failed: %v", err)
		}
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

	// Use the Replit public dev domain when available, else fall back to localhost.
	base := fmt.Sprintf("http://localhost:%d", port)
	if domain := os.Getenv("REPLIT_DEV_DOMAIN"); domain != "" {
		base = "https://" + domain
	}

	fmt.Println(bold + cyan + `
  ███████╗ ██████╗ ██████╗ ███████╗███████╗██████╗ 
     ███╔╝██╔═══██╗██╔══██╗   ███╔╝██╔════╝██╔══██╗
    ███╔╝ ██║   ██║██████╔╝  ███╔╝ █████╗  ██████╔╝
   ███╔╝  ██║   ██║██╔══██╗███╔╝   ██╔══╝  ██╔══██╗
  ███████╗╚██████╔╝██║  ██║███████╗███████╗██║  ██║
  ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝  ╚═╝
               L 7   S T R E S S E R` + reset)
	fmt.Println()
	fmt.Println(dim + "  ┌──────────────────────────────────────────────────────────────┐" + reset)
	fmt.Println(dim + "  │" + bold + cyan + "                    API SERVER READY                          " + reset + dim + "│" + reset)
	fmt.Println(dim + "  ├──────────────────────────────────────────────────────────────┤" + reset)
	fmt.Printf(dim+"  │"+reset+"  "+green+"POST"+reset+"  %-56s"+dim+"│"+reset+"\n", base+"/api/attack/start")
	fmt.Printf(dim+"  │"+reset+"  "+yellow+"POST"+reset+"  %-56s"+dim+"│"+reset+"\n", base+"/api/attack/stop")
	fmt.Printf(dim+"  │"+reset+"  "+cyan+"GET"+reset+"   %-56s"+dim+"│"+reset+"\n", base+"/api/attack/status")
	fmt.Printf(dim+"  │"+reset+"  "+cyan+"GET"+reset+"   %-56s"+dim+"│"+reset+"\n", base+"/api/health")
	fmt.Println(dim + "  └──────────────────────────────────────────────────────────────┘" + reset)
	fmt.Println()
}
