package input

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

type Spec struct {
	Title   string
	Version string
	Routes  []Route
	raw     *openapi3.T
}

type Route struct {
	Method  string
	Path    string
	Summary string
	Tags    []string
	Params  []Parameter
	Body    *SchemaRef
}

type Parameter struct {
	Name     string
	In       string
	Required bool
	Type     string
}

type SchemaRef struct {
	Type       string
	Properties map[string]*SchemaRef
	Items      *SchemaRef
}

func LoadOpenAPI(data []byte) (*Spec, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	spec := &Spec{
		Title:   doc.Info.Title,
		Version: doc.Info.Version,
		Routes:  make([]Route, 0),
		raw:     doc,
	}

	// Iterate through paths
	paths := doc.Paths.Map()
	for path, pathItem := range paths {
		if pathItem.Get != nil {
			spec.Routes = append(spec.Routes, routeFromOp("GET", path, pathItem.Get))
		}
		if pathItem.Post != nil {
			spec.Routes = append(spec.Routes, routeFromOp("POST", path, pathItem.Post))
		}
		if pathItem.Put != nil {
			spec.Routes = append(spec.Routes, routeFromOp("PUT", path, pathItem.Put))
		}
		if pathItem.Delete != nil {
			spec.Routes = append(spec.Routes, routeFromOp("DELETE", path, pathItem.Delete))
		}
		if pathItem.Patch != nil {
			spec.Routes = append(spec.Routes, routeFromOp("PATCH", path, pathItem.Patch))
		}
		if pathItem.Head != nil {
			spec.Routes = append(spec.Routes, routeFromOp("HEAD", path, pathItem.Head))
		}
		if pathItem.Options != nil {
			spec.Routes = append(spec.Routes, routeFromOp("OPTIONS", path, pathItem.Options))
		}
	}

	return spec, nil
}

func routeFromOp(method, path string, op *openapi3.Operation) Route {
	route := Route{
		Method:  method,
		Path:    path,
		Summary: op.Summary,
		Tags:    op.Tags,
		Params:  make([]Parameter, 0),
	}

	// Parameters
	if op.Parameters != nil {
		for _, p := range op.Parameters {
			param := Parameter{
				Name:     p.Value.Name,
				In:       p.Value.In,
				Required: p.Value.Required,
			}
			route.Params = append(route.Params, param)
		}
	}

	return route
}

func (s *Spec) RoutesForMethod(method string) []string {
	var routes []string
	for _, r := range s.Routes {
		if r.Method == method {
			routes = append(routes, r.Path)
		}
	}
	return routes
}

func (s *Spec) DocFor(method, path string) string {
	for _, r := range s.Routes {
		if r.Method == method && r.Path == path {
			doc := fmt.Sprintf("%s %s\n", r.Method, r.Path)
			if r.Summary != "" {
				doc += r.Summary + "\n"
			}
			if len(r.Params) > 0 {
				doc += "\nParameters:\n"
				for _, p := range r.Params {
					required := ""
					if p.Required {
						required = " (required)"
					}
					doc += fmt.Sprintf("  - %s (%s%s)\n", p.Name, p.In, required)
				}
			}
			return doc
		}
	}
	return ""
}
