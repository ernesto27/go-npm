package manager

import (
	"encoding/json"
	"fmt"
	"npm-packager/manifest"
	"os"
)

type ParseJsonManifest struct {
}

func newParseJsonManifest() *ParseJsonManifest {
	return &ParseJsonManifest{}
}

func (p *ParseJsonManifest) parse(filePath string) (*manifest.NPMPackage, error) {
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
