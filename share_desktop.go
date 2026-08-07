//go:build !js || !wasm

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// The desktop share client talks to the same share-service HTTP API. The
// service origin defaults to a local dev server and can be overridden with
// the QSIM_SHARE_URL environment variable, e.g.
//   QSIM_SHARE_URL=https://api.qsim.example.com ./qsim

func desktopShareBaseURL() string {
	if u := os.Getenv("QSIM_SHARE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://localhost:8080"
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

func desktopShareUpload(data string) (string, error) {
	body, err := json.Marshal(map[string]string{"data": data})
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Post(desktopShareBaseURL()+sharePath, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("share service returned %s", resp.Status)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("sharepoint returned no id")
	}
	return desktopShareBaseURL() + "/?share=" + out.ID, nil
}

func desktopShareFetch(id string) (string, error) {
	resp, err := httpClient.Get(desktopShareBaseURL() + sharePath + "/" + id)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("share endpoint returned %s", resp.Status)
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Data == "" {
		return "", errNotFound
	}
	return out.Data, nil
}

// shareIDFromURL is a no-op on desktop (no page URL to parse).
func shareIDFromURL() string {
	return ""
}