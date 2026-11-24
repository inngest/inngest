// Package icebergcatalog is an embedded Iceberg REST catalog server, a thin
// HTTP layer over iceberg-go's SQL catalog (sqlite/postgres via bun). It
// exists so the devserver/self-host persistent DuckDB mode can ATTACH an
// Iceberg catalog (DuckDB can only write Iceberg through a REST, Glue or S3
// Tables catalog) w/o running Lakekeeper locally. Open starts one on
// 127.0.0.1 w/ a bearer token.
//
// The wire format is a spec-compliant subset of the Iceberg REST catalog API
// (https://github.com/apache/iceberg/blob/main/open-api/rest-catalog-open-api.yaml),
// nothing Inngest-specific, so cloud can point the same ATTACH at Lakekeeper.
//
// TODO(auth): the bearer token check is a single shared static token (the
// same one we put in the DuckDB iceberg SECRET). No /v1/oauth/tokens, no
// scoping. Fine for 127.0.0.1, not for anything exposed.
//
// Credentials: w/ Options.Credentials set, create/load vend storage creds to
// a client that sends `X-Iceberg-Access-Delegation: vended-credentials`, and
// GET .../credentials refreshes them. W/o it (local disk), clients bring
// their own.
//
// TODO(maintenance): no orphan file cleanup or compaction. Commits expire
// old snapshots and delete metadata.json files that fell out of the
// metadata log (see Retention), but every DuckDB commit's manifest list +
// manifest(s) stay behind forever.
//
// TODO(views): /views endpoints aren't routed even though the sql catalog
// supports views. DuckDB doesn't need them (yet).
//
// TODO(txn): POST /v1/transactions/commit (multi-table) returns 406, the sql
// catalog doesn't implement catalog.TransactionalCatalog in v0.7.0.
//
// TODO(metrics): POST .../tables/{table}/metrics is accepted and ignored.
package icebergcatalog

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	icebergio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table"
)

// namespaceSeparator is the spec's multi-level namespace separator in path
// params (%1F, the unit separator).
const namespaceSeparator = "\x1f"

// Catalog is the subset of iceberg-go catalog functionality we need. The sql
// catalog (`catalog/sql.Catalog`) satisfies it.
type Catalog interface {
	catalog.Catalog
}

// Options configures a Server.
type Options struct {
	// Warehouse is the default warehouse location (local dir or s3:// URI).
	// Tables w/o an explicit location go under
	// `<Warehouse>/<ns>.db/<table>`, matching iceberg-go's own default.
	Warehouse string
	// Token, when non-empty, requires `Authorization: Bearer <Token>` on
	// every request. See TODO(auth).
	Token string
	// Logger, when non-nil, gets one line per request.
	Logger *slog.Logger
	// Retention configures snapshot expiry + metadata file cleanup on
	// commit. The zero value is on w/ the defaults.
	Retention Retention
	// OnRequest, when non-nil, is called after every request (used by tests
	// to record what DuckDB actually calls).
	OnRequest func(method, path string, status int)
	// CatalogDB, when non-nil, is the database the sql catalog behind cat
	// was created on. It turns on the load/commit fast path (see
	// fastPath), which skips re-reading metadata.json on every request.
	CatalogDB *CatalogDB
	// Credentials, when non-nil, vends storage credentials to clients that
	// ask for them (see CredentialVendor). W/o it, clients need their own.
	Credentials CredentialVendor
	// ReadOnly rejects every request that could change the catalog (all
	// but GET/HEAD and metrics reports), IE for a reader whose duckdb
	// holds the token but should never commit.
	ReadOnly bool
}

// Server is an Iceberg REST catalog backed by a catalog.Catalog.
type Server struct {
	cat    Catalog
	opts   Options
	mux    *http.ServeMux
	tables tableCache
	// fast is nil w/o Options.CatalogDB.
	fast *fastPath
	// io loads per-operation FileIO instances when fast is nil (for remote GC/purge).
	io ioLoader
	// now is the clock snapshot ages are measured against (tests swap it).
	now func() time.Time
}

// New returns a Server. The caller owns cat (and its *sql.DB).
func New(cat Catalog, opts Options) *Server {
	if opts.ReadOnly && opts.Credentials != nil {
		opts.Credentials = readOnlyCredentials(opts.Credentials)
	}
	s := &Server{cat: cat, opts: opts, mux: http.NewServeMux(), fast: newFastPath(opts.CatalogDB), now: time.Now}
	s.routes()
	return s
}

func (s *Server) getFileIO(ctx context.Context, loc string) (icebergio.IO, error) {
	if s.fast != nil {
		return s.fast.io.load(ctx, s.fast.props, loc)
	}
	var props iceberg.Properties
	if s.opts.CatalogDB != nil {
		props = s.opts.CatalogDB.Properties
	}
	return s.io.load(ctx, props, loc)
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
	"GET /v1/{prefix}/namespaces/{namespace}/tables/{table}/credentials",
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
	s.handle(metricsRoute, s.reportMetrics)
	s.handle("GET /v1/namespaces/{ns}/tables/{table}/credentials", s.loadCredentials)
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
			// The matched route (IE "GET /v1/namespaces/{ns}/tables/{table}"),
			// not the raw URI, so nothing the client sent ends up in the log.
			route := r.Pattern
			if route == "" {
				route = "unmatched"
			}
			s.opts.Logger.Debug("iceberg rest", "route", route, "status", rec.status, "dur", time.Since(start))
		}
	}()

	if s.opts.Token != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+s.opts.Token)) != 1 {
		writeError(rec, http.StatusUnauthorized, "NotAuthorizedException", "missing or invalid bearer token")
		return
	}
	if s.opts.ReadOnly && !s.readOnlyRequest(r) {
		writeError(rec, http.StatusForbidden, "ForbiddenException", "this catalog is read-only")
		return
	}
	s.mux.ServeHTTP(rec, r)
}

const metricsRoute = "POST /v1/namespaces/{ns}/tables/{table}/metrics"

// readOnlyRequest reports whether r can't change the catalog. A metrics
// report is a POST but only gets logged. NOTE: that's decided on the route
// r matches, not its path: a commit to a table named `metrics` ends in
// /metrics too.
func (s *Server) readOnlyRequest(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		return true
	case http.MethodPost:
		_, pattern := s.mux.Handler(r)
		return pattern == metricsRoute
	}
	return false
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

// Listen binds 127.0.0.1:0 and starts s in the background. It returns the
// base URL (`http://127.0.0.1:<port>`) to use as DuckDB's attach URI.
func (s *Server) Listen(ctx context.Context) (string, <-chan error, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	done := make(chan error, 1)
	go func() { done <- s.Start(ctx, l) }()
	return "http://" + l.Addr().String(), done, nil
}

// ---- wire types ----

type identifier struct {
	Namespace []string `json:"namespace"`
	Name      string   `json:"name"`
}

type configResponse struct {
	Defaults  iceberg.Properties `json:"defaults"`
	Overrides iceberg.Properties `json:"overrides"`
	Endpoints []string           `json:"endpoints,omitempty"`
}

type namespaceResponse struct {
	Namespace  []string           `json:"namespace"`
	Properties iceberg.Properties `json:"properties"`
}

type createNamespaceRequest struct {
	Namespace  []string           `json:"namespace"`
	Properties iceberg.Properties `json:"properties"`
}

type updateNamespacePropertiesRequest struct {
	Removals []string           `json:"removals"`
	Updates  iceberg.Properties `json:"updates"`
}

type createTableRequest struct {
	Name          string                 `json:"name"`
	Location      string                 `json:"location,omitempty"`
	Schema        *iceberg.Schema        `json:"schema"`
	PartitionSpec *iceberg.PartitionSpec `json:"partition-spec,omitempty"`
	WriteOrder    *table.SortOrder       `json:"write-order,omitempty"`
	StageCreate   bool                   `json:"stage-create,omitempty"`
	Properties    iceberg.Properties     `json:"properties,omitempty"`
}

type loadTableResponse struct {
	MetadataLocation string `json:"metadata-location,omitempty"`
	// Metadata is a table.Metadata, or its cached encoding
	// (json.RawMessage) on the load path.
	Metadata           any                 `json:"metadata"`
	Config             iceberg.Properties  `json:"config,omitempty"`
	StorageCredentials []storageCredential `json:"storage-credentials,omitempty"`
}

type commitTableRequest struct {
	Identifier   *identifier        `json:"identifier,omitempty"`
	Requirements table.Requirements `json:"requirements"`
	Updates      table.Updates      `json:"updates"`
}

type commitTableResponse struct {
	MetadataLocation string          `json:"metadata-location"`
	Metadata         json.RawMessage `json:"metadata"`
}

type renameTableRequest struct {
	Source      identifier `json:"source"`
	Destination identifier `json:"destination"`
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
	writeJSON(w, http.StatusOK, configResponse{
		Defaults:  iceberg.Properties{},
		Overrides: iceberg.Properties{},
		Endpoints: endpoints,
	})
}

func (s *Server) listNamespaces(w http.ResponseWriter, r *http.Request) {
	var parent table.Identifier
	if p := r.URL.Query().Get("parent"); p != "" {
		parent = strings.Split(p, namespaceSeparator)
	}
	nss, err := s.cat.ListNamespaces(r.Context(), parent)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	if nss == nil {
		nss = []table.Identifier{}
	}
	// TODO: pagination (pageToken/pageSize). Not needed at devserver scale.
	writeJSON(w, http.StatusOK, map[string]any{"namespaces": nss})
}

func (s *Server) createNamespace(w http.ResponseWriter, r *http.Request) {
	var req createNamespaceRequest
	if !readJSON(w, r, &req) {
		return
	}
	if err := validateNames(req.Namespace); err != nil {
		writeCatalogError(w, err)
		return
	}
	if req.Properties == nil {
		req.Properties = iceberg.Properties{}
	}
	if err := s.checkLocationProps(req.Properties); err != nil {
		writeCatalogError(w, err)
		return
	}
	if err := s.cat.CreateNamespace(r.Context(), req.Namespace, req.Properties); err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, namespaceResponse(req))
}

func (s *Server) getNamespace(w http.ResponseWriter, r *http.Request) {
	ns := pathNamespace(r)
	props, err := s.cat.LoadNamespaceProperties(r.Context(), ns)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	if props == nil {
		props = iceberg.Properties{}
	}
	writeJSON(w, http.StatusOK, namespaceResponse{Namespace: ns, Properties: props})
}

func (s *Server) headNamespace(w http.ResponseWriter, r *http.Request) {
	ok, err := s.cat.CheckNamespaceExists(r.Context(), pathNamespace(r))
	writeHead(w, ok, err)
}

func (s *Server) dropNamespace(w http.ResponseWriter, r *http.Request) {
	if err := s.cat.DropNamespace(r.Context(), pathNamespace(r)); err != nil {
		writeCatalogError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateNamespaceProperties(w http.ResponseWriter, r *http.Request) {
	var req updateNamespacePropertiesRequest
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.checkLocationProps(req.Updates); err != nil {
		writeCatalogError(w, err)
		return
	}
	sum, err := s.cat.UpdateNamespaceProperties(r.Context(), pathNamespace(r), req.Removals, req.Updates)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (s *Server) listTables(w http.ResponseWriter, r *http.Request) {
	ids := []identifier{}
	for id, err := range s.cat.ListTables(r.Context(), pathNamespace(r)) {
		if err != nil {
			writeCatalogError(w, err)
			return
		}
		ids = append(ids, toIdentifier(id))
	}
	writeJSON(w, http.StatusOK, map[string]any{"identifiers": ids})
}

func (s *Server) createTable(w http.ResponseWriter, r *http.Request) {
	ns := pathNamespace(r)
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
	if req.Location != "" {
		loc, err := s.checkLocation(req.Location)
		if err != nil {
			writeCatalogError(w, err)
			return
		}
		req.Location = loc
	}
	if err := s.checkLocationProps(req.Properties); err != nil {
		writeCatalogError(w, err)
		return
	}
	ident := append(append(table.Identifier{}, ns...), req.Name)

	if req.StageCreate {
		// Stage-create: build fresh metadata but don't persist anything. The
		// client follows up w/ a commit carrying assert-create + the full
		// update list, which iceberg-go's CommitTable handles as a create
		// (current == nil).
		s.stageCreate(w, r, ident, req)
		return
	}

	// W/o an explicit location, derive the same default iceberg-go's
	// CreateTable would and pass it explicitly, so it goes through
	// checkLocation (which resolves symlinks) before anything is written
	// there.
	if req.Location == "" {
		loc, err := s.defaultLocation(r.Context(), ns, req.Name)
		if err != nil {
			writeCatalogError(w, err)
			return
		}
		req.Location = loc
	}
	if err := s.locations().checkMetadataDir(req.Location, req.Properties); err != nil {
		writeCatalogError(w, err)
		return
	}
	opts := []catalog.CreateTableOpt{catalog.WithLocation(req.Location)}
	if req.PartitionSpec != nil {
		opts = append(opts, catalog.WithPartitionSpec(req.PartitionSpec))
	}
	if req.WriteOrder != nil {
		opts = append(opts, catalog.WithSortOrder(*req.WriteOrder))
	}
	if req.Properties != nil {
		opts = append(opts, catalog.WithProperties(req.Properties))
	}
	tbl, err := s.cat.CreateTable(r.Context(), ident, req.Schema, opts...)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	cfg, creds, err := s.tableCredentials(r, ident, tbl.Location())
	if err != nil {
		writeVendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loadTableResponse{
		MetadataLocation:   tbl.MetadataLocation(),
		Metadata:           tbl.Metadata(),
		Config:             cfg,
		StorageCredentials: creds,
	})
}

func (s *Server) stageCreate(w http.ResponseWriter, r *http.Request, ident table.Identifier, req createTableRequest) {
	ctx := r.Context()
	ns := catalog.NamespaceFromIdent(ident)
	if ok, err := s.cat.CheckNamespaceExists(ctx, ns); err != nil || !ok {
		if err == nil {
			err = fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, strings.Join(ns, "."))
		}
		writeCatalogError(w, err)
		return
	}
	if ok, err := s.cat.CheckTableExists(ctx, ident); err != nil || ok {
		if err == nil {
			err = fmt.Errorf("%w: %s", catalog.ErrTableAlreadyExists, strings.Join(ident, "."))
		}
		writeCatalogError(w, err)
		return
	}
	loc := req.Location
	if loc == "" {
		var err error
		loc, err = s.defaultLocation(ctx, ns, req.Name)
		if err != nil {
			writeCatalogError(w, err)
			return
		}
	}
	spec := req.PartitionSpec
	if spec == nil {
		spec = iceberg.UnpartitionedSpec
	}
	order := table.UnsortedSortOrder
	if req.WriteOrder != nil {
		order = *req.WriteOrder
	}
	md, err := table.NewMetadata(req.Schema, spec, order, loc, req.Properties)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BadRequestException", err.Error())
		return
	}
	cfg, creds, err := s.tableCredentials(r, ident, loc)
	if err != nil {
		writeVendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loadTableResponse{Metadata: md, Config: cfg, StorageCredentials: creds})
}

// defaultLocation mirrors iceberg-go's internal getDefaultWarehouseLocation
// (catalog/internal/utils.go), which isn't exported.
func (s *Server) defaultLocation(ctx context.Context, ns table.Identifier, name string) (string, error) {
	props, err := s.cat.LoadNamespaceProperties(ctx, ns)
	if err != nil {
		return "", err
	}
	var loc string
	switch {
	case props.Get("location", "") != "":
		loc, err = url.JoinPath(props.Get("location", ""), name)
	case s.opts.Warehouse == "":
		return "", errors.New("no warehouse configured and no location given")
	default:
		loc, err = url.JoinPath(s.opts.Warehouse, strings.Join(ns, ".")+".db", name)
	}
	if err != nil {
		return "", err
	}
	// Names and the namespace location are validated on the way in, this
	// just makes sure nothing slipped through (IE a pre-existing property).
	return s.checkLocation(loc)
}

func (s *Server) loadTable(w http.ResponseWriter, r *http.Request) {
	// `snapshots=refs` is ignored, we always return all snapshots.
	ident := pathTable(r)
	if c, ok := s.cachedCurrent(r.Context(), ident); ok {
		// NOTE: the table's own location, not c.loc's parent: the two differ
		// w/ `write.metadata.path` set or after a set-location commit.
		cfg, creds, err := s.tableCredentials(r, ident, c.md.Location())
		if err != nil {
			writeVendError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, loadTableResponse{
			MetadataLocation:   c.loc,
			Metadata:           c.body,
			Config:             cfg,
			StorageCredentials: creds,
		})
		return
	}
	tbl, err := s.cat.LoadTable(r.Context(), ident)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	body, err := s.tables.body(ident, tbl.MetadataLocation(), tbl.Metadata())
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	cfg, creds, err := s.tableCredentials(r, ident, tbl.Location())
	if err != nil {
		writeVendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loadTableResponse{
		MetadataLocation:   tbl.MetadataLocation(),
		Metadata:           body,
		Config:             cfg,
		StorageCredentials: creds,
	})
}

func (s *Server) headTable(w http.ResponseWriter, r *http.Request) {
	ok, err := s.cat.CheckTableExists(r.Context(), pathTable(r))
	writeHead(w, ok, err)
}

func (s *Server) commitTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ident := pathTable(r)
	var raw struct {
		Requirements json.RawMessage `json:"requirements"`
		Updates      json.RawMessage `json:"updates"`
	}
	if !readJSON(w, r, &raw) {
		return
	}
	var req commitTableRequest
	if err := json.Unmarshal(raw.Requirements, &req.Requirements); err != nil {
		writeError(w, http.StatusBadRequest, "BadRequestException", "malformed requirements: "+err.Error())
		return
	}
	updates, err := fixCreateUpdates(req.Requirements, raw.Updates)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BadRequestException", "malformed updates: "+err.Error())
		return
	}
	// Decoded twice: checkCommit applies its copy to a throwaway builder.
	var checkUpdates table.Updates
	if err := json.Unmarshal(updates, &req.Updates); err != nil {
		writeError(w, http.StatusBadRequest, "BadRequestException", "malformed updates: "+err.Error())
		return
	}
	if err := json.Unmarshal(updates, &checkUpdates); err != nil {
		writeError(w, http.StatusBadRequest, "BadRequestException", "malformed updates: "+err.Error())
		return
	}

	// iceberg-go's requirement Validate errors are plain errors.New, not
	// wrapped in table.ErrCommitFailed, so pre-validate here to get a clean
	// 409. CommitTable re-validates and CASes on metadata_location, so a
	// concurrent commit between here and there still loses (and that path
	// does wrap ErrCommitFailed). The requirement-failed-inside-CommitTable
	// window is caught by isRequirementFailure below.
	//
	// The base is the cached version when there is one: duckdb loads the
	// table right before writing, so it's almost always current, and a
	// stale one can't let a bad commit through (CommitTable re-validates
	// against what's actually stored). A requirement failure against it is
	// re-checked against a fresh load so a stale cache never fakes a 409.
	//
	// W/ the fast path the cached version is checked against the catalog
	// db first, and if it's current (and the requirements hold) the commit
	// skips the sql catalog entirely (see fastPath).
	var (
		current  table.Metadata
		cached   cachedTable
		verified bool
	)
	fromCache := false
	if s.fast != nil {
		cached, verified = s.cachedCurrent(ctx, ident)
		fromCache = verified
	} else {
		cached, fromCache = s.tables.get(ident)
	}
	if fromCache {
		current = cached.md
	}
	if !fromCache || !requirementsHold(req.Requirements, current) {
		verified = false
		current = nil
		if tbl, err := s.cat.LoadTable(ctx, ident); err == nil {
			current = tbl.Metadata()
		} else if !errors.Is(err, catalog.ErrNoSuchTable) {
			writeCatalogError(w, err)
			return
		}
	}
	for _, req := range req.Requirements {
		if err := req.Validate(current); err != nil {
			writeError(w, http.StatusConflict, "CommitFailedException", err.Error())
			return
		}
	}
	if current == nil && !hasAssertCreate(req.Requirements) {
		writeCatalogError(w, fmt.Errorf("%w: %s", catalog.ErrNoSuchTable, strings.Join(ident, ".")))
		return
	}
	next, err := s.checkCommit(current, checkUpdates)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	// Ours go after the client's updates are validated, and only ever
	// remove snapshots (see retain).
	s.retain(current, next, checkUpdates, &req, s.now())

	var (
		md  table.Metadata
		loc string
	)
	if verified {
		md, loc, err = s.fast.commit(ctx, ident, cached, req.Requirements, req.Updates)
	} else {
		md, loc, err = s.cat.CommitTable(ctx, ident, req.Requirements, req.Updates)
	}
	if err != nil {
		s.tables.drop(ident)
		writeCatalogError(w, err)
		return
	}
	s.deleteOldMetadata(ctx, current, md, loc)
	body, err := s.tables.put(ident, loc, md)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, commitTableResponse{MetadataLocation: loc, Metadata: body})
}

func (s *Server) dropTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ident := pathTable(r)
	defer s.tables.drop(ident)
	if r.URL.Query().Get("purgeRequested") != "true" {
		if err := s.cat.DropTable(ctx, ident); err != nil {
			writeCatalogError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// NOTE: purge doesn't use iceberg-go's PurgeTable, which deletes every
	// file the table's manifests reference wherever it is (and manifests are
	// written by the client, so we can't vouch for their contents). This
	// only ever deletes the table's own directory, after checking it's
	// strictly inside the warehouse.
	tbl, err := s.cat.LoadTable(ctx, ident)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	loc, local, err := s.purgeLocation(tbl.Metadata())
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	if err := s.cat.DropTable(ctx, ident); err != nil {
		writeCatalogError(w, err)
		return
	}
	if loc != "" {
		// Table directories aren't exclusive (a rename keeps the directory,
		// so a new table w/ the old name gets the same default location), so
		// skip the purge if any remaining table lives in, above or below it.
		if shared, err := s.locationInUse(ctx, loc); err != nil || shared {
			if s.opts.Logger != nil {
				s.opts.Logger.Warn("dropped iceberg table w/o purging files (location shared w/ another table)", "table", strings.Join(ident, "."), "location", loc, "error", err)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if local {
			if err := os.RemoveAll(loc); err != nil && s.opts.Logger != nil {
				s.opts.Logger.Warn("dropped iceberg table but failed to purge its files", "table", strings.Join(ident, "."), "error", err)
			}
		} else {
			s.purgeRemoteTable(ctx, ident, loc, tbl.MetadataLocation(), tbl.Metadata())
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// purgeRemoteTable removes a dropped table's files from a remote warehouse
// (e.g. s3://) via FileIO. It prefers RemoveAllIO, then ListableIO +
// BulkRemovableIO/WriteFileIO.Remove, and falls back to removing known
// metadata/manifest files under loc via WriteFileIO.Remove.
func (s *Server) purgeRemoteTable(ctx context.Context, ident table.Identifier, loc, metaLoc string, md table.Metadata) {
	fsys, err := s.getFileIO(ctx, loc)
	if err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("dropped iceberg table but failed to load FileIO for purge", "table", strings.Join(ident, "."), "location", loc, "error", err)
		}
		return
	}
	if rmAll, ok := fsys.(icebergio.RemoveAllIO); ok {
		if err := rmAll.RemoveAll(loc); err != nil && !errors.Is(err, fs.ErrNotExist) && s.opts.Logger != nil {
			s.opts.Logger.Warn("dropped iceberg table but failed to purge its files", "table", strings.Join(ident, "."), "location", loc, "error", err)
		}
		return
	}
	wfs, ok := fsys.(icebergio.WriteFileIO)
	if !ok {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("dropped iceberg table w/o purging files (FileIO does not support writing)", "table", strings.Join(ident, "."), "location", loc)
		}
		return
	}

	c := s.locations()
	var paths []string
	if listable, ok := fsys.(icebergio.ListableIO); ok {
		walkErr := listable.WalkDir(loc, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d != nil && d.IsDir() {
				return nil
			}
			if c.checkUnder(p, []string{loc}) == nil {
				paths = append(paths, p)
			}
			return nil
		})
		if walkErr != nil && !errors.Is(walkErr, fs.ErrNotExist) && s.opts.Logger != nil {
			s.opts.Logger.Warn("failed to list remote iceberg table files for purge", "table", strings.Join(ident, "."), "location", loc, "error", walkErr)
		}
	} else {
		seen := map[string]bool{}
		add := func(p string) {
			if p == "" || seen[p] {
				return
			}
			if clean, err := c.check(p); err == nil && c.checkUnder(clean, []string{loc}) == nil {
				seen[p] = true
				paths = append(paths, clean)
			}
		}
		add(metaLoc)
		if md != nil {
			for e := range md.PreviousFiles() {
				add(e.MetadataFile)
			}
			for _, p := range metadataPaths(md) {
				add(p)
			}
		}
	}
	if len(paths) == 0 {
		return
	}
	if brm, ok := fsys.(icebergio.BulkRemovableIO); ok {
		if _, err := brm.DeleteFiles(ctx, paths); err != nil && s.opts.Logger != nil {
			s.opts.Logger.Warn("dropped iceberg table but failed to purge some remote files", "table", strings.Join(ident, "."), "location", loc, "error", err)
		}
		return
	}
	for _, p := range paths {
		if err := wfs.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) && s.opts.Logger != nil {
			s.opts.Logger.Warn("dropped iceberg table but failed to purge remote file", "table", strings.Join(ident, "."), "path", p, "error", err)
		}
	}
}

func (s *Server) renameTable(w http.ResponseWriter, r *http.Request) {
	var req renameTableRequest
	if !readJSON(w, r, &req) {
		return
	}
	from := append(append(table.Identifier{}, req.Source.Namespace...), req.Source.Name)
	to := append(append(table.Identifier{}, req.Destination.Namespace...), req.Destination.Name)
	for _, id := range []table.Identifier{from, to} {
		if err := validateNames(id); err != nil {
			writeCatalogError(w, err)
			return
		}
	}
	defer s.tables.drop(from, to)
	if _, err := s.cat.RenameTable(r.Context(), from, to); err != nil {
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

func pathNamespace(r *http.Request) table.Identifier {
	return strings.Split(r.PathValue("ns"), namespaceSeparator)
}

func pathTable(r *http.Request) table.Identifier {
	return append(pathNamespace(r), r.PathValue("table"))
}

func toIdentifier(id table.Identifier) identifier {
	return identifier{
		Namespace: catalog.NamespaceFromIdent(id),
		Name:      catalog.TableNameFromIdent(id),
	}
}

// fixCreateUpdates works around an iceberg-go v0.7.0 bug in the
// stage-create -> commit(assert-create) path. CommitTable starts a new table
// from `table.NewMetadata(iceberg.NewSchema(0), ...)`, so the base metadata
// already has an *empty* schema 0 (and spec 0 / sort order 0). The client's
// `add-schema` (schema-id 0) then gets reassigned to schema 1, and its
// `set-current-schema` 0 points at the empty placeholder, so the table ends
// up w/ 0 columns. Same hazard for partitioned specs / real sort orders.
//
// For create commits we rewrite set-current-schema / set-default-spec /
// set-default-sort-order to -1 ("last added") when a matching add-* precedes
// them, which is what the client meant anyways. The placeholder empty schema
// 0 still ends up in `schemas` (harmless, but ugly).
//
// TODO: fix upstream (base a create commit on an empty builder instead of
// `table.NewMetadata(iceberg.NewSchema(0), ...)` in catalog/sql's
// CommitTable) and drop this.
func fixCreateUpdates(reqs table.Requirements, raw json.RawMessage) (json.RawMessage, error) {
	if !hasAssertCreate(reqs) || len(raw) == 0 {
		return raw, nil
	}
	var ups []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &ups); err != nil {
		return nil, err
	}
	setters := map[string]struct{ add, idField string }{
		"set-current-schema":     {"add-schema", "schema-id"},
		"set-default-spec":       {"add-spec", "spec-id"},
		"set-default-sort-order": {"add-sort-order", "sort-order-id"},
	}
	added := map[string]bool{}
	for _, u := range ups {
		var action string
		_ = json.Unmarshal(u["action"], &action)
		added[action] = true
		if st, ok := setters[action]; ok && added[st.add] {
			u[st.idField] = json.RawMessage("-1")
		}
	}
	return json.Marshal(ups)
}

func hasAssertCreate(reqs table.Requirements) bool {
	for _, r := range reqs {
		if r.GetType() == "assert-create" {
			return true
		}
	}
	return false
}

const maxRequestBodyBytes = 32 << 20 // 32 MiB

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "RequestEntityTooLargeException", "request body too large")
			return false
		}
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

// classify maps iceberg-go errors onto the REST spec's status codes and
// exception type names (the names are what Java/pyiceberg clients key off).
func classify(err error) (int, string) {
	switch {
	case errors.Is(err, catalog.ErrNoSuchTable):
		return http.StatusNotFound, "NoSuchTableException"
	case errors.Is(err, catalog.ErrNoSuchNamespace):
		return http.StatusNotFound, "NoSuchNamespaceException"
	case errors.Is(err, catalog.ErrNoSuchView):
		return http.StatusNotFound, "NoSuchViewException"
	case errors.Is(err, catalog.ErrTableAlreadyExists),
		errors.Is(err, catalog.ErrNamespaceAlreadyExists),
		errors.Is(err, catalog.ErrViewAlreadyExists):
		return http.StatusConflict, "AlreadyExistsException"
	case errors.Is(err, catalog.ErrNamespaceNotEmpty):
		return http.StatusConflict, "NamespaceNotEmptyException"
	case errors.Is(err, table.ErrCommitFailed), isRequirementFailure(err):
		return http.StatusConflict, "CommitFailedException"
	case errors.Is(err, errBadRequest),
		errors.Is(err, catalog.ErrInvalidIdentifier),
		errors.Is(err, table.ErrInvalidRequirement),
		errors.Is(err, table.ErrInvalidMetadata):
		return http.StatusBadRequest, "BadRequestException"
	case errors.Is(err, iceberg.ErrNotImplemented):
		return http.StatusNotAcceptable, "UnsupportedOperationException"
	default:
		return http.StatusInternalServerError, "InternalServerError"
	}
}

// isRequirementFailure matches iceberg-go v0.7.0's unwrapped requirement
// errors ("Requirement failed: ..." / "requirement failed: ..."). Gross, but
// there's no sentinel to errors.Is against.
func isRequirementFailure(err error) bool {
	return strings.HasPrefix(strings.ToLower(err.Error()), "requirement failed")
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// requirementsHold reports whether every requirement validates against md.
func requirementsHold(reqs []table.Requirement, md table.Metadata) bool {
	for _, r := range reqs {
		if r.Validate(md) != nil {
			return false
		}
	}
	return true
}

// cachedCurrent is ident's cached metadata if the fast path is on and the
// catalog db says it's still the current version.
func (s *Server) cachedCurrent(ctx context.Context, ident table.Identifier) (cachedTable, bool) {
	if s.fast == nil {
		return cachedTable{}, false
	}
	c, ok := s.tables.get(ident)
	if !ok {
		return cachedTable{}, false
	}
	loc, ok, err := s.fast.currentLocation(ctx, ident)
	if err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("iceberg catalog fast path: checking metadata location", "table", strings.Join(ident, "."), "error", err)
		}
		return cachedTable{}, false
	}
	if !ok || loc != c.loc || parseMetadataVersion(loc) < 0 {
		return cachedTable{}, false
	}
	return c, true
}
