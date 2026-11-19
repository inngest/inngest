package icebergcatalog

import (
	"context"
	"net/http"
	"strings"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/table"
)

// CredentialVendor hands out storage credentials for one table's files, IE
// short-lived S3 creds scoped to location. They go back to the client as
// FileIO properties (`s3.access-key-id`, `s3.secret-access-key`,
// `s3.session-token`, `s3.session-token-expires-at-ms`, `s3.region`,
// `s3.endpoint`, `s3.path-style-access`), which is what duckdb's iceberg
// extension reads. A client refreshes them through
// GET .../tables/{table}/credentials once they expire.
type CredentialVendor interface {
	VendCredentials(ctx context.Context, ident table.Identifier, location string) (iceberg.Properties, error)
}

// accessDelegationHeader is how a client asks for vended credentials (vs
// remote signing, which we don't support).
const accessDelegationHeader = "X-Iceberg-Access-Delegation"

type storageCredential struct {
	Prefix string             `json:"prefix"`
	Config iceberg.Properties `json:"config"`
}

type loadCredentialsResponse struct {
	StorageCredentials []storageCredential `json:"storage-credentials"`
}

func wantsVendedCredentials(r *http.Request) bool {
	for _, v := range r.Header.Values(accessDelegationHeader) {
		for _, mode := range strings.Split(v, ",") {
			if strings.TrimSpace(mode) == "vended-credentials" {
				return true
			}
		}
	}
	return false
}

// tableCredentials vends creds for a create/load response when the client
// asked for them and a vendor is configured. NOTE: the creds go in both
// `storage-credentials` (what newer clients, duckdb included, prefer) and
// `config` (the older spot every client still reads).
func (s *Server) tableCredentials(r *http.Request, ident table.Identifier, location string) (iceberg.Properties, []storageCredential, error) {
	if s.opts.Credentials == nil || !wantsVendedCredentials(r) {
		return iceberg.Properties{}, nil, nil
	}
	props, err := s.opts.Credentials.VendCredentials(r.Context(), ident, location)
	if err != nil {
		return nil, nil, err
	}
	return props, []storageCredential{{Prefix: location, Config: props}}, nil
}

func (s *Server) loadCredentials(w http.ResponseWriter, r *http.Request) {
	if s.opts.Credentials == nil {
		writeError(w, http.StatusNotFound, "NotFoundException", "this catalog doesn't vend credentials")
		return
	}
	ident := pathTable(r)
	tbl, err := s.cat.LoadTable(r.Context(), ident)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	loc := tbl.Location()
	props, err := s.opts.Credentials.VendCredentials(r.Context(), ident, loc)
	if err != nil {
		writeVendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loadCredentialsResponse{
		StorageCredentials: []storageCredential{{Prefix: loc, Config: props}},
	})
}

func writeVendError(w http.ResponseWriter, err error) {
	writeError(w, http.StatusServiceUnavailable, "ServiceUnavailableException", "vending credentials: "+err.Error())
}
