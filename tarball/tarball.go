package tarball

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/ernesto27/go-npm/integrity"
	"github.com/ernesto27/go-npm/utils"
)

type Tarball struct {
	TarballPath string
	validator   *integrity.Validator
}

func NewTarball(tarballPath string) *Tarball {
	return &Tarball{
		TarballPath: tarballPath,
		validator:   integrity.New(),
	}
}

func (d *Tarball) Download(url string) error {
	filename := path.Base(url)
	filePath := filepath.Join(d.TarballPath, filename)

	_, _, err := utils.DownloadFile(url, filePath, "")
	return err
}

// DownloadAs downloads a tarball from url and saves it with a custom filename
func (d *Tarball) DownloadAs(url, filename string) error {
	filePath := filepath.Join(d.TarballPath, filename)
	_, _, err := utils.DownloadFile(url, filePath, "")
	return err
}

// DownloadAndValidate downloads a tarball and validates its integrity hash
// Downloads to temp file first, validates, then finalizes only if valid
// Returns ErrNoIntegrity if integrityHash is empty (strict mode)
func (d *Tarball) DownloadAndValidate(url, filename, integrityHash string) error {
	if integrityHash == "" {
		return integrity.ErrNoIntegrity
	}

	filePath := filepath.Join(d.TarballPath, filename)
	tempPath := filePath + ".tmp"

	// Download to temp file
	_, _, err := utils.DownloadFile(url, tempPath, "")
	if err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("download failed: %w", err)
	}

	// Validate integrity before finalizing
	if err := d.validator.ValidateFileStrict(tempPath, integrityHash); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("integrity validation failed for %s: %w", filename, err)
	}

	// Atomic rename: only succeeds if validation passed
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to finalize download: %w", err)
	}

	return nil
}
