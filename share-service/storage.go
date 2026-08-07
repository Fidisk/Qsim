package main

import (
	"os"
	"path/filepath"
)

// store persists shared circuits by their id.
type store interface {
	save(id string, data string) error
	load(id string) (string, error)
}

// openStore prepares the storage backend (no-op for disk).
func openStore(st store) error {
	if o, ok := st.(interface{ open() error }); ok {
		return o.open()
	}
	return nil
}

// makeStore builds the backend named by STORAGE env (default "disk").
func makeStore(backend string) (store, error) {
	switch backend {
	case "disk":
		dir := os.Getenv("QSIM_SHARES_DIR")
		if dir == "" {
			dir = "./data"
		}
		return &diskStore{dir: dir}, nil
	case "s3":
		return makeS3Store()
	default:
		return nil, osMkdirErr(backend)
	}
}

func osMkdirErr(backend string) error {
	return &unknownStoreError{backend: backend}
}

type unknownStoreError struct{ backend string }

func (e *unknownStoreError) Error() string {
	return "unknown STORAGE=" + e.backend + " (available: disk, s3)"
}

// diskStore keeps one file per share id inside a local directory.
type diskStore struct {
	dir string
}

func (d *diskStore) open() error {
	return os.MkdirAll(d.dir, 0o755)
}

func (d *diskStore) save(id string, data string) error {
	return os.WriteFile(filepath.Join(d.dir, id), []byte(data), 0o644)
}

func (d *diskStore) load(id string) (string, error) {
	b, err := os.ReadFile(filepath.Join(d.dir, id))
	if err != nil {
		return "", err
	}
	return string(b), nil
}