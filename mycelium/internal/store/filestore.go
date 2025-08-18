package store

import (
	"fmt"
	"os"
	"path"
    "path/filepath"
	"strings"

	"mycelium/internal/crawler"

	"github.com/google/uuid"
)

type FileStore struct {
	outDirectory string
}

func NewFileStore(outDirectory string) *FileStore {
	return &FileStore{
		outDirectory: outDirectory,
	}
}

func (fs *FileStore) Store(item crawler.StoreItem, extension string) (string, error) {
	data, err := item.Marshal()
	if err != nil {
		return "", fmt.Errorf("failed to marshal store item: %w", err)
	}
    prefix := item.Prefix()
	id := uuid.New()
	idStr := id.String()
	out := path.Join(fs.outDirectory, prefix, idStr+strings.ToLower(extension))

    if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
        return "", fmt.Errorf("failed to create directories: %w", err)
    }
	if err := os.WriteFile(out, data, 0755); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", out, err)
	}

	return idStr, nil
}

func (fs *FileStore) Retrieve(id string, extension string) ([]byte, error) {
	file := path.Join(fs.outDirectory, id+strings.ToLower(extension))
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve file %s: %w", file, err)
	}
	return data, nil
}
