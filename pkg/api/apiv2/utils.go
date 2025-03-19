package apiv2

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/syscode"
)

func writeEmpty(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeFailBody(
	ctx context.Context,
	w http.ResponseWriter,
	err error,
	statusCode ...int,
) {
	var serr syscode.Error
	if errors.Is(err, sql.ErrNoRows) {
		serr = syscode.Error{Code: syscode.CodeNotFound}
	} else {
		serr = syscode.FromError(err)
	}

	l := logger.StdlibLogger(ctx)
	status := http.StatusInternalServerError
	if len(statusCode) > 0 {
		status = statusCode[0]
	} else if serr.Code == syscode.CodeNotFound {
		status = http.StatusNotFound
	}

	msg := http.StatusText(status)
	if status == http.StatusInternalServerError {
		l.Error(msg, "error", serr)
	} else if serr.Message != "" {
		msg = serr.Message
	}

	resBody := responseError{
		Code:    serr.Code,
		Message: msg,
	}

	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resBody)
}

func appFromCQRS(app *cqrs.App) App {
	var archivedAt *time.Time
	if app.ArchivedAt.Valid {
		archivedAt = &app.ArchivedAt.Time
	}

	return App{
		ArchivedAt: archivedAt,
		CreatedAt:  app.CreatedAt,
		ID:         app.Name,
		InternalID: app.ID,
	}
}
