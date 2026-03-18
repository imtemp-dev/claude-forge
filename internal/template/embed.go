package template

import (
	"embed"
	"io/fs"
)

//go:embed all:templates
var embeddedRaw embed.FS

// EmbeddedTemplates returns the embedded template filesystem
// with the "templates/" prefix stripped.
func EmbeddedTemplates() (fs.FS, error) {
	return fs.Sub(embeddedRaw, "templates")
}
