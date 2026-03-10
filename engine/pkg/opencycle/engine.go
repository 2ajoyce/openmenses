// Package opencycle exposes the public API surface for the opencycle engine.
// External consumers (mobile wrappers, dev CLI, integration tests) interact
// with the engine through the [Engine] type rather than the internal packages.
package opencycle

import (
	"context"
	"fmt"
	"net/http"

	"github.com/2ajoyce/opencycle/engine/internal/service"
	"github.com/2ajoyce/opencycle/engine/internal/storage"
	"github.com/2ajoyce/opencycle/engine/internal/storage/memory"
	"github.com/2ajoyce/opencycle/engine/internal/storage/sqlite"
	"github.com/2ajoyce/opencycle/gen/go/opencycle/v1/opencyclev1connect"
)

// Engine is the top-level object that wires together storage, validation, and
// the Connect-RPC service handler. Create one with [NewEngine].
type Engine struct {
	store   storage.Repository
	svc     *service.CycleTrackerService
	closeFn func() error // non-nil only for SQLite stores
}

// options holds the configuration built up by Option functions.
type options struct {
	sqlitePath string // empty → in-memory
}

// Option is a functional option for [NewEngine].
type Option func(*options)

// WithSQLite configures the engine to use a SQLite database at the given path.
// Use ":memory:" for a private in-memory SQLite database (distinct from the
// default pure-Go in-memory backend, but useful for schema testing).
func WithSQLite(path string) Option {
	return func(o *options) {
		o.sqlitePath = path
	}
}

// WithInMemory configures the engine to use the pure-Go in-memory backend.
// This is the default when no storage option is provided and is the recommended
// backend for unit and integration tests.
func WithInMemory() Option {
	return func(o *options) {
		o.sqlitePath = ""
	}
}

// NewEngine constructs an Engine with the supplied options.
//
// If no storage option is provided the engine defaults to the in-memory
// backend. Call [Engine.Close] when done to release any resources.
func NewEngine(ctx context.Context, opts ...Option) (*Engine, error) {
	cfg := &options{}
	for _, o := range opts {
		o(cfg)
	}

	var (
		store   storage.Repository
		closeFn func() error
	)

	if cfg.sqlitePath != "" {
		s, err := sqlite.Open(ctx, cfg.sqlitePath)
		if err != nil {
			return nil, fmt.Errorf("opencycle: open sqlite: %w", err)
		}
		store = s
		closeFn = s.Close
	} else {
		store = memory.New()
		closeFn = func() error { return nil }
	}

	svc, err := service.New(store)
	if err != nil {
		if closeFn != nil {
			closeFn() //nolint:errcheck
		}
		return nil, fmt.Errorf("opencycle: init service: %w", err)
	}

	return &Engine{store: store, svc: svc, closeFn: closeFn}, nil
}

// Handler returns the HTTP path prefix and handler for the Connect-RPC service.
// Mount the handler at the returned path on any [http.ServeMux]:
//
//	path, handler := engine.Handler()
//	mux.Handle(path, handler)
func (e *Engine) Handler() (string, http.Handler) {
	return opencyclev1connect.NewCycleTrackerServiceHandler(e.svc)
}

// Close releases any resources held by the engine (e.g., SQLite file handles).
// After Close returns the engine must not be used.
func (e *Engine) Close() error {
	if e.closeFn != nil {
		return e.closeFn()
	}
	return nil
}
