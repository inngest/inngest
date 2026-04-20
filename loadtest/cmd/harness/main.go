// Command harness runs the Inngest load-test harness: a standalone HTTP
// server that serves a UI, manages SDK worker subprocesses, and persists
// run results into SQLite.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/inngest/inngest/loadtest/internal/api"
	"github.com/inngest/inngest/loadtest/internal/runner"
	"github.com/inngest/inngest/loadtest/internal/storage"
	"github.com/inngest/inngest/loadtest/internal/uiembed"
)

func main() {
	var (
		port      = flag.Int("port", 9010, "HTTP port")
		dbPath    = flag.String("db", "loadtest.db", "SQLite DB path")
		workerBin = flag.String("worker", "./bin/worker-go", "Path to compiled worker-go binary")
		socketDir = flag.String("socket-dir", "", "Directory for unix sockets (default: os.TempDir)")
	)
	flag.Parse()

	if *socketDir == "" {
		*socketDir = os.TempDir()
	}
	if _, err := os.Stat(*workerBin); err != nil {
		log.Fatalf("worker binary %s not found: %v", *workerBin, err)
	}
	absWorker, err := filepath.Abs(*workerBin)
	if err != nil {
		log.Fatalf("resolve worker path: %v", err)
	}

	store, err := storage.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	hostID, _ := os.Hostname()
	mgr := runner.NewManager(store, runner.Options{
		WorkerBin: absWorker,
		SocketDir: *socketDir,
		BasePort:  40000,
	})

	mux := http.NewServeMux()
	apiH := api.New(store, mgr, hostID)
	apiH.Mount(mux)
	mux.Handle("/", uiembed.SPAHandler())

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	go func() {
		log.Printf("harness listening on http://127.0.0.1:%d", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()
	<-ctx.Done()
	log.Printf("shutting down...")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
}

func logRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
