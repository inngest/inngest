package apiv2

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/headers"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
)

const (
	// maxRequestSize is the maximum request size in bytes.
	maxRequestSize = 1024 * 1024
)

type NewOpts struct {
	AppSvc   appSvc
	BasePath string
	EnvSvc   envSvc
}

func (o *NewOpts) Validate() error {
	var err error
	if o.AppSvc == nil {
		err = errors.Join(err, fmt.Errorf("AppSvc is required"))
	}
	if o.BasePath == "" {
		err = errors.Join(err, fmt.Errorf("BasePath is required"))
	}
	if o.EnvSvc == nil {
		err = errors.Join(err, fmt.Errorf("EnvSvc is required"))
	}
	return err
}

func New(
	ctx context.Context,
	o NewOpts,
) (http.Handler, error) {
	err := o.Validate()
	if err != nil {
		return nil, err
	}

	api := api{
		Router: chi.NewRouter(),
		opts:   o,
	}

	swagger, err := GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("error getting swagger: %w", err)
	}
	swagger.Servers = []*openapi3.Server{{URL: o.BasePath}}

	api.Group(func(r chi.Router) {
		r.Use(middleware.RequestSize(maxRequestSize))
		r.Use(headers.ContentTypeJsonResponse())

		r.Group(func(r chi.Router) {
			r.Use(oapimiddleware.OapiRequestValidatorWithOptions(
				swagger,
				&oapimiddleware.Options{
					ErrorHandler: func(
						w http.ResponseWriter,
						message string,
						statusCode int,
					) {
						writeFailBody(ctx, w, fmt.Errorf("%s", message), statusCode)
					},
				},
			))

			HandlerWithOptions(api, ChiServerOptions{BaseRouter: r})
		})
	})

	return api, nil
}

type api struct {
	chi.Router

	opts NewOpts
}
