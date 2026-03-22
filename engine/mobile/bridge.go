// Package mobile provides gomobile-compatible bindings for the openmenses
// engine. This package is designed to be built with `gomobile bind`, which
// generates native iOS/Android frameworks from Go code.
//
// gomobile has strict limitations on exported signatures: parameters and
// return values must be primitives (int, string, bool, error, []byte).
// No context.Context, interfaces, channels, or custom structs are allowed
// in exported function signatures.
//
// The bridge manages the engine lifecycle, HTTP server, authentication, and
// static file serving for the bundled UI. See [Start] and [Stop].
package mobile

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	openmenses "github.com/2ajoyce/openmenses/engine/pkg/openmenses"
)

// state holds the active engine and HTTP server. Access is protected by mu.
type state struct {
	eng    *openmenses.Engine
	srv    *http.Server
	ln     net.Listener
	token  string
	cancel context.CancelFunc
}

var (
	mu      sync.Mutex
	running *state
)

// Start initializes the engine and starts the HTTP server. Call [Port] and
// [AuthToken] after a successful Start to retrieve the assigned port and
// auth token. Call [Stop] to shut down.
//
// dbPath: absolute path to SQLite database file (will be created if absent).
// uiAssetsDir: absolute path to directory containing the built UI files
//
//	(index.html, JS, CSS, etc.). Pass empty string to skip static
//	file serving (useful for tests that only need the API).
//
// gomobile restriction: exported functions may return at most one value plus
// an optional error. Port and token are exposed via separate getter functions.
func Start(dbPath string, uiAssetsDir string) error {
	mu.Lock()
	defer mu.Unlock()

	if running != nil {
		return fmt.Errorf("engine already running; call Stop() first")
	}

	// Generate auth token: 32 random bytes → 64-char hex string.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generate auth token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Create context with cancel so we can shut down gracefully.
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize engine with SQLite backend.
	eng, err := openmenses.NewEngine(ctx, openmenses.WithSQLite(dbPath))
	if err != nil {
		cancel()
		return fmt.Errorf("initialize engine: %w", err)
	}

	// Build HTTP mux with service handler and optional static file server.
	mux := http.NewServeMux()

	servicePath, serviceHandler := eng.Handler()
	mux.Handle(servicePath, authMiddleware(token, serviceHandler))

	if uiAssetsDir != "" {
		mux.Handle("/", spaFileServer(uiAssetsDir))
	}

	// Listen on a random port (OS assigns port 0 → random available port).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		eng.Close() //nolint:errcheck
		return fmt.Errorf("listen: %w", err)
	}

	srv := &http.Server{Handler: mux}

	// Start server in a goroutine.
	go func() {
		_ = srv.Serve(ln)
	}()

	// Store state.
	running = &state{
		eng:    eng,
		srv:    srv,
		ln:     ln,
		token:  token,
		cancel: cancel,
	}

	return nil
}

// Port returns the TCP port the engine's HTTP server is listening on.
// Returns 0 if the engine is not running.
func Port() int {
	mu.Lock()
	defer mu.Unlock()
	if running == nil {
		return 0
	}
	return running.ln.Addr().(*net.TCPAddr).Port
}

// AuthToken returns the bearer token required for Connect-RPC requests.
// Returns an empty string if the engine is not running.
func AuthToken() string {
	mu.Lock()
	defer mu.Unlock()
	if running == nil {
		return ""
	}
	return running.token
}

// Stop gracefully shuts down the HTTP server and closes the engine.
// Safe to call multiple times. Returns an error if shutdown fails (but
// continues trying to clean up).
func Stop() error {
	mu.Lock()
	defer mu.Unlock()

	if running == nil {
		return nil // Idempotent: already stopped.
	}

	s := running
	running = nil

	// Gracefully shut down HTTP server.
	if err := s.srv.Shutdown(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "engine bridge: shutdown error: %v\n", err)
		// Continue cleanup despite shutdown error
	}

	// Close engine (releases SQLite file handles, etc).
	if err := s.eng.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "engine bridge: close error: %v\n", err)
		// Continue cleanup despite close error
	}

	// Cancel context.
	s.cancel()

	return nil
}

// authMiddleware wraps a handler and rejects requests missing a valid
// "Authorization: Bearer <token>" header. Requests without Auth are rejected
// with HTTP 401 Unauthorized.
func authMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("missing Authorization header"))
			return
		}

		const prefix = "Bearer "
		if len(auth) < len(prefix) || auth[:len(prefix)] != prefix {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("invalid Authorization header"))
			return
		}

		providedToken := auth[len(prefix):]
		if providedToken != token {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("invalid token"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// spaFileServer serves static files from dir, falling back to index.html for
// paths that don't match a real file. This enables client-side SPA routing
// where the framework handles unknown routes.
//
// Uses http.Dir which is path-traversal safe: it rejects any path that would
// escape the root directory (e.g. requests like /../../../../etc/passwd).
func spaFileServer(dir string) http.Handler {
	root := http.Dir(dir)
	fileServer := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.Dir.Open is path-traversal safe.
		f, err := root.Open(r.URL.Path)
		if err != nil {
			// File doesn't exist or path is not accessible — serve index.html.
			indexFile, indexErr := root.Open("/index.html")
			if indexErr != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			indexFile.Close() //nolint:errcheck

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		defer f.Close() //nolint:errcheck

		// Check if the path is a directory — fall back to its index.html or SPA root.
		stat, err := f.Stat()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if stat.IsDir() {
			dirIndex, indexErr := root.Open(r.URL.Path + "/index.html")
			if indexErr != nil {
				// No index in subdirectory — fall back to SPA root index.html.
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				http.ServeFile(w, r, filepath.Join(dir, "index.html"))
				return
			}
			dirIndex.Close() //nolint:errcheck
		}

		fileServer.ServeHTTP(w, r)
	})
}
