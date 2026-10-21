package pages

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// Shapes is the set of coolshapes shape identifiers ("star-1", "flower-3"…).
type Shapes map[string]bool

// Has reports whether id is a known shape.
func (s Shapes) Has(id string) bool {
	return s[id]
}

// LoadShapes lists the shapes/*.svg files of the built front: the files are the
// single source of truth, so a removed shape is refused instead of shown broken.
func LoadShapes(fsys fs.FS) (Shapes, error) {
	files, err := fs.Glob(fsys, "shapes/*.svg")
	if err != nil {
		return nil, fmt.Errorf("list shapes: %w", err)
	}
	shapes := make(Shapes, len(files))
	for _, f := range files {
		shapes[strings.TrimSuffix(path.Base(f), ".svg")] = true
	}
	return shapes, nil
}
