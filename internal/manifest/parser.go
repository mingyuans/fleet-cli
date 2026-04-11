package manifest

import (
	"encoding/xml"
	"fmt"
	"os"
)

// ParseFile parses a manifest XML file and returns a Manifest struct.
func ParseFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}
	return Parse(data)
}

// Parse parses manifest XML bytes and returns a Manifest struct.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := xml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest XML: %w", err)
	}
	return &m, nil
}
