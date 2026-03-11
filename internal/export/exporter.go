package export

import (
	"fmt"

	"httpclient/internal/model"
)

type Exporter interface {
	Format() string
	Export(session *model.Session, env *model.Environment) ([]byte, error)
}

var exporters = make(map[string]func() Exporter)

func Register(name string, fn func() Exporter) {
	exporters[name] = fn
}

func Get(format string) (Exporter, error) {
	fn, ok := exporters[format]
	if !ok {
		return nil, fmt.Errorf("unknown format: %s (available: json, curl, har, http, bruno)", format)
	}
	return fn(), nil
}

func ListFormats() []string {
	formats := make([]string, 0, len(exporters))
	for f := range exporters {
		formats = append(formats, f)
	}
	return formats
}
