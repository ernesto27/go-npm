package packagejson

import (
	"encoding/json"
	"fmt"
	"go-npm/config"
	"os"
)

type PackageJSON struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Version      any               `json:"version"`
	Author       any               `json:"author"`
	Contributors any               `json:"contributors"`
	License      any               `json:"license"`
	Repository   any               `json:"repository"`
	Homepage     any               `json:"homepage"`
	Funding      any               `json:"funding"`
	Keywords     any               `json:"keywords"`
	Dependencies map[string]string `json:"dependencies"`
	Engines      any               `json:"engines"`
	Files        any               `json:"files"`
	Scripts      map[string]string `json:"scripts"`
	Main         any               `json:"main"`
	Bin          any               `json:"bin"`
	Types        string            `json:"types"`
	Exports      any               `json:"exports"`
	Private      bool              `json:"private"`
	Workspaces   any               `json:"workspaces"`
}

type PackageJSONParser struct {
	Config          *config.Config
	PackageJSON     *PackageJSON
	FilePath        string
	OriginalContent []byte
}

func NewPackageJSONParser(cfg *config.Config) *PackageJSONParser {
	return &PackageJSONParser{
		Config: cfg,
	}
}

func (p *PackageJSONParser) Parse(filePath string) (*PackageJSON, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var packageJSON PackageJSON
	if err := json.Unmarshal(fileContent, &packageJSON); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from file %s: %w", filePath, err)
	}

	p.PackageJSON = &packageJSON
	p.FilePath = filePath
	p.OriginalContent = fileContent

	return &packageJSON, nil
}
