// Package icebergcatalog is an embedded Iceberg REST catalog server. It
// exists so the devserver/self-host persistent DuckDB mode can ATTACH an
// Iceberg catalog (DuckDB can only write Iceberg through a REST, Glue or S3
// Tables catalog) w/o running Lakekeeper locally. Open starts one on
// 127.0.0.1 w/ a bearer token.
//
// It owns a small table metadata model (metadata.go, commit.go) instead of
// linking apache/iceberg-go, which drags arrow, substrait, bun + every SQL
// dialect and the aws sdk into the binary. Catalog state lives in sqlite or
// postgres in the standard JDBC catalog layout (store.go). iceberg-go is
// still used, but only by the tests (its REST client + conformance suite).
//
// The wire format is a spec-compliant subset of the Iceberg REST catalog API
// (https://github.com/apache/iceberg/blob/main/open-api/rest-catalog-open-api.yaml),
// nothing Inngest-specific, so cloud can point the same ATTACH at Lakekeeper.
//
// TODO(auth): the bearer token check is a single shared static token (the
// same one we put in the DuckDB iceberg SECRET). No /v1/oauth/tokens, no
// scoping. Fine for 127.0.0.1, not for anything exposed.
//
// TODO(creds): no credential vending (storage-credentials / config in
// LoadTable). Not needed for local disk.
//
// TODO(s3): only local warehouses. Metadata files are written w/ os.*, an
// S3 warehouse needs a small put/get/delete client.
//
// TODO(maintenance): no snapshot expiry, orphan file cleanup, or metadata
// file cleanup (write.metadata.delete-after-commit.enabled is ignored).
// Every DuckDB commit leaves a metadata.json + manifest list + manifest(s)
// behind forever.
//
// TODO(views): /views endpoints aren't routed. DuckDB doesn't need them
// (yet).
//
// TODO(txn): POST /v1/transactions/commit (multi-table) returns 406.
//
// TODO(metrics): POST .../tables/{table}/metrics is accepted and ignored.
package icebergcatalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// namespaceSeparator is the spec's multi-level namespace separator in path
// params (%1F, the unit separator).
const namespaceSeparator = "\x1f"

// CatalogName is the name stored in the JDBC layout's `catalog_name`
// column. It only matters if several catalogs share one backing database.
const CatalogName = "inngest"

// Options configures a Server.
type Options struct {
	// Warehouse is the warehouse directory (absolute). Tables w/o an
	// explicit location go under `<Warehouse>/<ns>.db/<table>`, matching
	// iceberg-go/Java's default.
	Warehouse string
	// CatalogName defaults to CatalogName.
	CatalogName string
	// Token, when non-empty, requires `Authorization: Bearer <Token>` on
	// every request. See TODO(auth).
	Token string
	// Logger, when non-nil, gets one line per request.
	Logger *slog.Logger
	// OnRequest, when non-nil, is called after every request (used by tests
	// to record what DuckDB actually calls).
	OnRequest func(method, path string, status int)
}

// Server is an Iceberg REST catalog backed by a database/sql db.
type Server struct {
	store *store
	opts  Options
	mux   *http.ServeMux
	now   func() time.Time
}

// New returns a Server, creating the catalog tables in db if needed. The
// caller owns db.
func New(ctx context.Context, db *sql.DB, opts Options) (*Server, error) {
	if opts.CatalogName == "" {
		opts.CatalogName = CatalogName
	}
	st, err := newStore(ctx, db, opts.CatalogName)
	if err != nil {
		return nil, err
	}
	s := &Server{store: st, opts: opts, mux: http.NewServeMux(), now: time.Now}
	s.routes()
	return s, nil
}

// endpoints is advertised in GET /v1/config so clients know what we support.
var endpoints = []string{
	"GET /v1/{prefix}/namespaces",
	"POST /v1/{prefix}/namespaces",
	"GET /v1/{prefix}/namespaces/{namespace}",
	"HEAD /v1/{prefix}/namespaces/{namespace}",
	"DELETE /v1/{prefix}/namespaces/{namespace}",
	"POST /v1/{prefix}/namespaces/{namespace}/properties",
	"GET /v1/{prefix}/namespaces/{namespace}/tables",
	"POST /v1/{prefix}/namespaces/{namespace}/tables",
	"GET /v1/{prefix}/namespaces/{namespace}/tables/{table}",
	"HEAD /v1/{prefix}/namespaces/{namespace}/tables/{table}",
	"POST /v1/{prefix}/namespaces/{namespace}/tables/{table}",
	"DELETE /v1/{prefix}/namespaces/{namespace}/tables/{table}",
	"POST /v1/{prefix}/tables/rename",
	"POST /v1/{prefix}/namespaces/{namespace}/tables/{table}/metrics",
}

// handle registers h behind validation of the `{ns}`/`{table}` path params,
// so no handler ever sees a name that could escape the warehouse once it's
// joined into a location.
func (s *Server) handle(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if ns := r.PathValue("ns"); ns != "" {
			if err := validateNames(strings.Split(ns, namespaceSeparator)); err != nil {
				writeCatalogError(w, err)
				return
			}
		}
		if t := r.PathValue("table"); t != "" {
			if err := validateName(t); err != nil {
				writeCatalogError(w, err)
				return
			}
		}
		h(w, r)
	})
}

func (s *Server) routes() {
	// NOTE: we don't use a {prefix} segment. Config returns no `prefix`
	// override so clients build `/v1/namespaces/...` directly. Go's mux can't
	// register both `/v1/{prefix}/namespaces` and `/v1/namespaces/{ns}`
	// (ambiguous), and we don't need multi-warehouse.
	s.mux.HandleFunc("GET /v1/config", s.getConfig)

	s.mux.HandleFunc("GET /v1/namespaces", s.listNamespaces)
	s.mux.HandleFunc("POST /v1/namespaces", s.createNamespace)
	s.handle("GET /v1/namespaces/{ns}", s.getNamespace)
	s.handle("HEAD /v1/namespaces/{ns}", s.headNamespace)
	s.handle("DELETE /v1/namespaces/{ns}", s.dropNamespace)
	s.handle("POST /v1/namespaces/{ns}/properties", s.updateNamespaceProperties)

	s.handle("GET /v1/namespaces/{ns}/tables", s.listTables)
	s.handle("POST /v1/namespaces/{ns}/tables", s.createTable)
	s.handle("GET /v1/namespaces/{ns}/tables/{table}", s.loadTable)
	s.handle("HEAD /v1/namespaces/{ns}/tables/{table}", s.headTable)
	s.handle("POST /v1/namespaces/{ns}/tables/{table}", s.commitTable)
	s.handle("DELETE /v1/namespaces/{ns}/tables/{table}", s.dropTable)
	s.handle("POST /v1/namespaces/{ns}/tables/{table}/metrics", s.reportMetrics)
	s.mux.HandleFunc("POST /v1/tables/rename", s.renameTable)

	s.mux.HandleFunc("POST /v1/transactions/commit", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotAcceptable, "UnsupportedOperationException", "multi-table transactions are not supported")
	})
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	start := time.Now()
	defer func() {
		if s.opts.OnRequest != nil {
			s.opts.OnRequest(r.Method, r.URL.RequestURI(), rec.status)
		}
		if s.opts.Logger != nil {
			s.opts.Logger.Debug("iceberg rest", "method", r.Method, "path", r.URL.RequestURI(), "status", rec.status, "dur", time.Since(start))
		}
	}()

	if s.opts.Token != "" && r.Header.Get("Authorization") != "Bearer "+s.opts.Token {
		writeError(rec, http.StatusUnauthorized, "NotAuthorizedException", "missing or invalid bearer token")
		return
	}
	s.mux.ServeHTTP(rec, r)
}

// Start serves s on l until ctx is done, then shuts down. Pass a listener
// bound to 127.0.0.1:0 and read the port back from l.Addr().
func (s *Server) Start(ctx context.Context, l net.Listener) error {
	srv := &http.Server{Handler: s, ReadHeaderTimeout: 10 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(l) }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		<-errc
		return nil
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// ---- wire types ----

type identifier struct {
	Namespace []string `json:"namespace"`
	Name      string   `json:"name"`
}

func (id identifier) String() string {
	return strings.Join(append(append([]string{}, id.Namespace...), id.Name), ".")
}

type namespaceResponse struct {
	Namespace  []string          `json:"namespace"`
	Properties map[string]string `json:"properties"`
}

type createTableRequest struct {
	Name          string            `json:"name"`
	Location      string            `json:"location,omitempty"`
	Schema        jsonObject        `json:"schema"`
	PartitionSpec jsonObject        `json:"partition-spec,omitempty"`
	WriteOrder    jsonObject        `json:"write-order,omitempty"`
	StageCreate   bool              `json:"stage-create,omitempty"`
	Properties    map[string]string `json:"properties,omitempty"`
}

type loadTableResponse struct {
	MetadataLocation string            `json:"metadata-location,omitempty"`
	Metadata         *tableMetadata    `json:"metadata"`
	Config           map[string]string `json:"config"`
}

type errorModel struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

// ---- handlers ----

func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	// `warehouse` query param is accepted and ignored: one warehouse per
	// server.
	writeJSON(w, http.StatusOK, map[string]any{
		"defaults":  map[string]string{},
		"overrides": map[string]string{},
		"endpoints": endpoints,
	})
}

func (s *Server) listNamespaces(w http.ResponseWriter, r *http.Request) {
	var parent []string
	if p := r.URL.Query().Get("parent"); p != "" {
		parent = strings.Split(p, namespaceSeparator)
	}
	nss, err := s.store.listNamespaces(r.Context(), parent)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	// TODO: pagination (pageToken/pageSize). Not needed at devserver scale.
	writeJSON(w, http.StatusOK, map[string]any{"namespaces": nss})
}

func (s *Server) createNamespace(w http.ResponseWriter, r *http.Request) {
	var req namespaceResponse
	if !readJSON(w, r, &req) {
		return
	}
	if err := validateNames(req.Namespace); err != nil {
		writeCatalogError(w, err)
		return
	}
	if req.Properties == nil {
		req.Properties = map[string]string{}
	}
	if err := s.checkLocationProps(req.Properties); err != nil {
		writeCatalogError(w, err)
		return
	}
	if err := s.store.createNamespace(r.Context(), req.Namespace, req.Properties); err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (s *Server) getNamespace(w http.ResponseWriter, r *http.Request) {
	ns := pathNamespace(r)
	props, err := s.store.loadNamespace(r.Context(), ns)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, namespaceResponse{Namespace: ns, Properties: props})
}

func (s *Server) headNamespace(w http.ResponseWriter, r *http.Request) {
	ok, err := s.store.namespaceExists(r.Context(), s.store.db, pathNamespace(r))
	writeHead(w, ok, err)
}

func (s *Server) dropNamespace(w http.ResponseWriter, r *http.Request) {
	if err := s.store.dropNamespace(r.Context(), pathNamespace(r)); err != nil {
		writeCatalogError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateNamespaceProperties(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Removals []string          `json:"removals"`
		Updates  map[string]string `json:"updates"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.checkLocationProps(req.Updates); err != nil {
		writeCatalogError(w, err)
		return
	}
	sum, err := s.store.updateNamespace(r.Context(), pathNamespace(r), req.Removals, req.Updates)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (s *Server) listTables(w http.ResponseWriter, r *http.Request) {
	ids, err := s.store.listTables(r.Context(), pathNamespace(r))
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"identifiers": ids})
}

func (s *Server) createTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req createTableRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.Name == "" || req.Schema == nil {
		writeError(w, http.StatusBadRequest, "BadRequestException", "name and schema are required")
		return
	}
	if err := validateName(req.Name); err != nil {
		writeCatalogError(w, err)
		return
	}
	id := identifier{Namespace: pathNamespace(r), Name: req.Name}
	if req.Location != "" {
		loc, err := s.checkLocation(req.Location)
		if err != nil {
			writeCatalogError(w, err)
			return
		}
		req.Location = loc
	}
	if req.Properties == nil {
		req.Properties = map[string]string{}
	}
	if err := s.checkLocationProps(req.Properties); err != nil {
		writeCatalogError(w, err)
		return
	}
	if ok, err := s.store.namespaceExists(ctx, s.store.db, id.Namespace); err != nil || !ok {
		if err == nil {
			err = fmt.Errorf("%w: %s", errNoSuchNamespace, strings.Join(id.Namespace, "."))
		}
		writeCatalogError(w, err)
		return
	}
	if _, err := s.store.tableLocation(ctx, id); err == nil {
		writeCatalogError(w, fmt.Errorf("%w: %s", errTableAlreadyExists, id))
		return
	} else if !errors.Is(err, errNoSuchTable) {
		writeCatalogError(w, err)
		return
	}

	updates, err := createUpdates(req)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	// A create is the same as a create commit (assert-create) w/ the
	// updates a client would send, so both go through the same builder and
	// checks.
	md, err := s.build(ctx, id, nil, "", updates)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	if req.StageCreate {
		// Stage-create: hand back fresh metadata but persist nothing. The
		// client follows up w/ a commit carrying assert-create + its own
		// full update list.
		writeJSON(w, http.StatusOK, loadTableResponse{Metadata: md, Config: map[string]string{}})
		return
	}
	loc, err := s.persist(ctx, id, nil, "", md)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loadTableResponse{MetadataLocation: loc, Metadata: md, Config: map[string]string{}})
}

// load returns a table's current metadata + its location.
func (s *Server) load(ctx context.Context, id identifier) (*tableMetadata, string, error) {
	loc, err := s.store.tableLocation(ctx, id)
	if err != nil {
		return nil, "", err
	}
	md, err := readMetadata(loc)
	if err != nil {
		return nil, "", fmt.Errorf("reading %s metadata: %w", id, err)
	}
	return md, loc, nil
}

func (s *Server) loadTable(w http.ResponseWriter, r *http.Request) {
	// `snapshots=refs` and X-Iceberg-Access-Delegation are ignored, we
	// always return all snapshots and never vend creds.
	md, loc, err := s.load(r.Context(), pathTable(r))
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loadTableResponse{MetadataLocation: loc, Metadata: md, Config: map[string]string{}})
}

func (s *Server) headTable(w http.ResponseWriter, r *http.Request) {
	_, err := s.store.tableLocation(r.Context(), pathTable(r))
	if errors.Is(err, errNoSuchTable) {
		writeHead(w, false, nil)
		return
	}
	writeHead(w, err == nil, err)
}

func (s *Server) commitTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := pathTable(r)
	var body struct {
		Identifier   *identifier       `json:"identifier,omitempty"`
		Requirements []json.RawMessage `json:"requirements"`
		Updates      []json.RawMessage `json:"updates"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	reqs := make([]requirement, 0, len(body.Requirements))
	create := false
	for _, raw := range body.Requirements {
		req, err := parseRequirement(raw)
		if err != nil {
			writeCatalogError(w, err)
			return
		}
		create = create || req.typ == "assert-create"
		reqs = append(reqs, req)
	}
	updates := make([]update, 0, len(body.Updates))
	for _, raw := range body.Updates {
		u, err := parseUpdate(raw)
		if err != nil {
			writeCatalogError(w, err)
			return
		}
		updates = append(updates, u)
	}

	current, loc, err := s.load(ctx, id)
	switch {
	case errors.Is(err, errNoSuchTable) && create:
		current = nil
	case err != nil:
		writeCatalogError(w, err)
		return
	}
	for _, req := range reqs {
		if err := req.check(current); err != nil {
			writeCatalogError(w, err)
			return
		}
	}
	md, err := s.build(ctx, id, current, loc, updates)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	if current != nil && len(updates) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"metadata-location": loc, "metadata": current})
		return
	}
	newLoc, err := s.persist(ctx, id, current, loc, md)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"metadata-location": newLoc, "metadata": md})
}

// build applies updates to current (nil for a create) and runs every check
// on the result.
func (s *Server) build(ctx context.Context, id identifier, current *tableMetadata, loc string, updates []update) (*tableMetadata, error) {
	b := newBuilder(current, loc, s.now())
	for _, u := range updates {
		if err := b.apply(u); err != nil {
			return nil, err
		}
	}
	md, err := b.finish(func() (string, error) { return s.defaultLocation(ctx, id) })
	if err != nil {
		return nil, err
	}
	if err := s.checkMetadata(md); err != nil {
		return nil, err
	}
	return md, nil
}

// persist writes md as the next metadata file and points the catalog at it
// (insert for a create, CAS from prevLoc otherwise). The file is removed
// again if the catalog update loses.
func (s *Server) persist(ctx context.Context, id identifier, current *tableMetadata, prevLoc string, md *tableMetadata) (string, error) {
	newLoc, err := s.writeMetadata(md, prevLoc)
	if err != nil {
		return "", err
	}
	if current == nil {
		err = s.store.createTable(ctx, id, newLoc)
	} else {
		err = s.store.swapTable(ctx, id, prevLoc, newLoc)
	}
	if err != nil {
		_ = os.Remove(newLoc)
		return "", err
	}
	return newLoc, nil
}

func (s *Server) dropTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := pathTable(r)
	if r.URL.Query().Get("purgeRequested") != "true" {
		if err := s.store.dropTable(ctx, id); err != nil {
			writeCatalogError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// NOTE: purge never deletes the files the table's manifests reference
	// (manifests are written by the client, so we can't vouch for their
	// contents). It only ever deletes the table's own directory, after
	// checking it's strictly inside the warehouse.
	md, _, err := s.load(ctx, id)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	loc, err := s.purgeLocation(md)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	if err := s.store.dropTable(ctx, id); err != nil {
		writeCatalogError(w, err)
		return
	}
	if err := os.RemoveAll(loc); err != nil && s.opts.Logger != nil {
		s.opts.Logger.Warn("dropped iceberg table but failed to purge its files", "table", id.String(), "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) renameTable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source      identifier `json:"source"`
		Destination identifier `json:"destination"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	for _, id := range []identifier{req.Source, req.Destination} {
		if err := validateNames(append(append([]string{}, id.Namespace...), id.Name)); err != nil {
			writeCatalogError(w, err)
			return
		}
	}
	if err := s.store.renameTable(r.Context(), req.Source, req.Destination); err != nil {
		writeCatalogError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reportMetrics(w http.ResponseWriter, r *http.Request) {
	// TODO(metrics): accept and ignore.
	w.WriteHeader(http.StatusNoContent)
}

// ---- helpers ----

func pathNamespace(r *http.Request) []string {
	return strings.Split(r.PathValue("ns"), namespaceSeparator)
}

func pathTable(r *http.Request) identifier {
	return identifier{Namespace: pathNamespace(r), Name: r.PathValue("table")}
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "BadRequestException", "malformed request: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, typ, msg string) {
	writeJSON(w, status, map[string]errorModel{"error": {Message: msg, Type: typ, Code: status}})
}

func writeHead(w http.ResponseWriter, ok bool, err error) {
	switch {
	case err != nil:
		status, _ := classify(err)
		w.WriteHeader(status)
	case ok:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func writeCatalogError(w http.ResponseWriter, err error) {
	status, typ := classify(err)
	writeError(w, status, typ, err.Error())
}

// classify maps our errors onto the REST spec's status codes and exception
// type names (the names are what Java/pyiceberg/iceberg-go clients key off).
func classify(err error) (int, string) {
	switch {
	case errors.Is(err, errNoSuchTable):
		return http.StatusNotFound, "NoSuchTableException"
	case errors.Is(err, errNoSuchNamespace):
		return http.StatusNotFound, "NoSuchNamespaceException"
	case errors.Is(err, errTableAlreadyExists), errors.Is(err, errNamespaceAlreadyExists):
		return http.StatusConflict, "AlreadyExistsException"
	case errors.Is(err, errNamespaceNotEmpty):
		return http.StatusConflict, "NamespaceNotEmptyException"
	case errors.Is(err, errCommitFailed):
		return http.StatusConflict, "CommitFailedException"
	case errors.Is(err, errBadRequest):
		return http.StatusBadRequest, "BadRequestException"
	default:
		return http.StatusInternalServerError, "InternalServerError"
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
