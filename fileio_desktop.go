//go:build !js || !wasm

package main

import (
	"os"
	"path/filepath"

	"github.com/sqweek/dialog"
)

const savesDir = "saves"

// Desktop saves live in a local "saves" directory next to the executable.

func platformListSaves() []string {
	_ = os.MkdirAll(savesDir, 0755)
	entries, err := os.ReadDir(savesDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".qsim" {
			names = append(names, e.Name())
		}
	}
	return names
}

func platformReadSave(name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(savesDir, name))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func platformWriteSave(name string, data string) error {
	_ = os.MkdirAll(savesDir, 0755)
	return os.WriteFile(filepath.Join(savesDir, name), []byte(data), 0644)
}

func platformRemoveSave(name string) error {
	return os.Remove(filepath.Join(savesDir, name))
}

func platformDownloadSave(name string, data string) {
	// The desktop already persists the file to disk; nothing to download.
}

func platformRequestSaveUpload(cb func(name string, data string)) {
	// Native file picker. If the dialog fails (e.g. no windowing system on a
	// headless server) we silently skip, keeping the Load window usable.
	filename, err := dialog.File().Filter("Qsim circuits", "qsim").Load()
	if err != nil || filename == "" {
		return
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	cb(filepath.Base(filename), string(data))
}

// Share API: compiled on desktop too, so the same build can publish links.
func platformShareUpload(data string) (string, error) {
	return desktopShareUpload(data)
}

func platformShareFetch(id string) (string, error) {
	return desktopShareFetch(id)
}

func platformShareBaseURL() string { return desktopShareBaseURL() }

// platformCopyToClipboard is a no-op on desktop: clipboard access needs an
// OS-specific binding that is out of scope here. The share window still shows
// the link so the user can copy it manually.
func platformCopyToClipboard(text string) {}