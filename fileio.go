package main

import (
	"errors"
)

// This file declares the platform-agnostic save/load/share API. Each runtime
// provides an implementation of the low-level helpers (see fileio_desktop.go
// and fileio_wasm.go).

// errNotFound is returned when a requested save or share does not exist.
var errNotFound = errors.New("not found")

// sharePath is the HTTP route of the share API on the share-service.
const sharePath = "/api/share"

// listSaves returns the names of available circuit saves for the platform.
func listSaves() []string { return platformListSaves() }

// readSave returns the serialized circuit content of a saved circuit.
func readSave(name string) (string, error) { return platformReadSave(name) }

// writeSave stores a serialized circuit under the given name.
func writeSave(name string, data string) error { return platformWriteSave(name, data) }

// removeSave deletes a saved circuit.
func removeSave(name string) error { return platformRemoveSave(name) }

// saveStateSerialized serializes the current circuit (same as SaveState).
func saveStateSerialized() string { return SaveState() }

// downloadSave lets the user download a circuit file to disk. On desktop the
// file already lives in the saves directory, so this is a no-op; on the web it
// triggers a browser download.
func downloadSave(name string, data string) {
	platformDownloadSave(name, data)
}

// requestSaveUpload opens a platform file-picker (web only) or does nothing.
// The callback receives the file name and its content when a file is chosen.
func requestSaveUpload(cb func(name string, data string)) {
	platformRequestSaveUpload(cb)
}

// shareCircuit uploads the serialized circuit and returns a shareable link.
func shareCircuit(data string) (string, error) { return platformShareUpload(data) }

// loadShare fetches a shared circuit's serialized content from its ID.
func loadShare(id string) (string, error) { return platformShareFetch(id) }

// shareBaseURL returns the origin of the share API (web: current origin).
func shareBaseURL() string { return platformShareBaseURL() }

// copyToClipboard writes text to the system clipboard (web only; no-op on
// desktop since there is no cross-platform clipboard binding here).
func copyToClipboard(text string) { platformCopyToClipboard(text) }