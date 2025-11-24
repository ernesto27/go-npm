package tarball

import (
	"fmt"
	"go-npm/manifest"
	"go-npm/utils"
	"os"
	"path"
	"path/filepath"
)

type Tarball struct {
	TarballPath string
}

func NewTarball() *Tarball {
	tarballPath := os.TempDir()
	return &Tarball{TarballPath: tarballPath}
}

func (d *Tarball) Download(version string, npmPackage manifest.NPMPackage) (string, error) {
	versionData, ok := npmPackage.Versions[version]
	if !ok {
		return "", fmt.Errorf("version %s not found in package %s", version, npmPackage.Name)
	}

	url := versionData.Dist.Tarball
	filename := path.Base(url)
	filePath := filepath.Join(d.TarballPath, filename)

	_, err := utils.DownloadFile(url, filePath)
	return filePath, err
}
