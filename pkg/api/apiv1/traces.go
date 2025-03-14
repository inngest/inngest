package apiv1

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/davecgh/go-spew/spew"
	coltrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func (a router) traces(w http.ResponseWriter, r *http.Request) {
	_, err := a.opts.AuthFinder(r.Context())
	if err != nil {
		respondError(w, r, http.StatusUnauthorized, "No auth found")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "Error reading body")
		return
	}

	req := &coltrace.ExportTraceServiceRequest{}
	isJSON := strings.Contains(r.Header.Get("Content-Type"), "json")
	if isJSON {
		err = protojson.Unmarshal(body, req)
	} else {
		err = proto.Unmarshal(body, req)
	}
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid payload")
		return
	}

	// TODO handle the spans
	spew.Dump(req.ResourceSpans)

	resp := &coltrace.ExportTraceServiceResponse{}
	var respBytes []byte
	if isJSON {
		respBytes, _ = protojson.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
	} else {
		respBytes, _ = proto.Marshal(resp)
		w.Header().Set("Content-Type", "application/x-protobuf")
	}

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write(respBytes)
		_ = gz.Close()
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func respondError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	isJSON := strings.Contains(r.Header.Get("Content-Type"), "json")
	status := &statuspb.Status{Message: msg}

	var data []byte
	if isJSON {
		data, _ = protojson.Marshal(status)
		w.Header().Set("Content-Type", "application/json")
	} else {
		data, _ = proto.Marshal(status)
		w.Header().Set("Content-Type", "application/x-protobuf")
	}

	w.WriteHeader(code)
	w.Write(data)
}
