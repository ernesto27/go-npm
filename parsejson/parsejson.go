package parsejson

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ernesto27/go-npm/manifest"
)

// Parser handles parsing of npm manifest JSON files
type Parser struct{}

// New creates a new Parser instance
func New() *Parser {
	return &Parser{}
}

// Parse reads and parses an npm manifest JSON file into an NPMPackage struct
func (p *Parser) Parse(filePath string) (*manifest.NPMPackage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	var npmPackage manifest.NPMPackage
	if err := json.NewDecoder(file).Decode(&npmPackage); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from file %s: %w", filePath, err)
	}

	return &npmPackage, nil
}
