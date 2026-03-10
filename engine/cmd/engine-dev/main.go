// Command engine-dev starts a local Connect-RPC server for development and
// manual testing of the openmenses engine.
//
// Usage:
//
//	engine-dev [--port 8080] [--db /path/to/db.sqlite]
//
// If --db is omitted the engine uses the pure-Go in-memory backend; data is
// lost when the process exits.  Pass --db :memory: to use an in-memory SQLite
// database (useful for testing schema migrations without touching the disk).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	openmenses "github.com/2ajoyce/openmenses/engine/pkg/openmenses"
)

func main() {
	port := flag.Int("port", 8080, "TCP port to listen on")
	db := flag.String("db", "", "SQLite database path; omit for in-memory backend")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var (
		eng *openmenses.Engine
		err error
	)

	if *db != "" {
		log.Printf("storage: sqlite  path=%s", *db)
		eng, err = openmenses.NewEngine(ctx, openmenses.WithSQLite(*db))
	} else {
		log.Printf("storage: in-memory (data will not persist)")
		eng, err = openmenses.NewEngine(ctx, openmenses.WithInMemory())
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine-dev: failed to initialise engine: %v\n", err)
		os.Exit(1)
	}
	defer eng.Close() //nolint:errcheck

	mux := http.NewServeMux()
	path, handler := eng.Handler()
	mux.Handle(path, handler)

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine-dev: listen %s: %v\n", addr, err)
		os.Exit(1)
	}

	srv := &http.Server{Handler: mux}

	log.Printf("engine-dev listening on http://%s", addr)
	log.Printf("Connect-RPC path prefix: %s", path)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		log.Println("engine-dev: shutting down…")
		if shutErr := srv.Shutdown(context.Background()); shutErr != nil {
			log.Printf("engine-dev: shutdown error: %v", shutErr)
		}
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "engine-dev: server error: %v\n", err)
			os.Exit(1)
		}
	}
}
