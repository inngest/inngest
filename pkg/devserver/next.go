package devserver

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"mime"
	"regexp"
)

//go:embed all:static
var static embed.FS

var parsedRoutes *routes

func init() {
	// Fix invalid mime type errors when loading JS from our assets on windows.
	_ = mime.AddExtensionType(".js", "application/javascript")
	parsedRoutes = &routes{}

	// Fetch the routes from the manifest.
	byt, err := static.ReadFile("static/routes-manifest.json")
	if err != nil {
		return
	}

	if err := json.Unmarshal(byt, parsedRoutes); err != nil {
		return
	}
	parsedRoutes.files = static
}

type routes struct {
	files embed.FS

	DynamicRoutes []*route
	StaticRoutes  []*route
}

// serve takes the current path, checks next routes, and loads the specific
// .html page for the given route.
func (r routes) serve(ctx context.Context, path string) []byte {
	for _, r := range r.StaticRoutes {
		if byt, err := r.match(ctx, path); err == nil {
			return byt
		}
	}
	for _, r := range r.DynamicRoutes {
		if byt, err := r.match(ctx, path); err == nil {
			return byt
		}
	}

	// Use index.html by default.
	byt, _ := parsedRoutes.files.ReadFile("static/index.html")
	return byt
}

type route struct {
	Page       string
	Regex      string
	NamedRegex string
	RouteKeys  map[string]string

	compiledRegex      *regexp.Regexp
	compiledNamedRegex *regexp.Regexp
}

func (r *route) match(ctx context.Context, path string) ([]byte, error) {
	if r.compiledRegex == nil {
		r.compiledRegex, _ = regexp.Compile(r.Regex)
		r.compiledNamedRegex, _ = regexp.Compile(r.NamedRegex)
	}

	if r.compiledRegex != nil && r.compiledRegex.MatchString(path) {
		file := "static" + r.Page + ".html"
		if r.Page == "/" {
			file = "static/index.html"
		}

		byt, err := parsedRoutes.files.ReadFile(file)
		if err == nil {
			return byt, nil
		}
	}
	return nil, fmt.Errorf("route does not match")
}
