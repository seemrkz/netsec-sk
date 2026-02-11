package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const appVersion = "0.1.0"

type serverInfo struct {
	URL       string `json:"url"`
	Port      int    `json:"port"`
	PID       int    `json:"pid"`
	StartedAt string `json:"started_at"`
	Version   string `json:"version"`
}

type healthResponse struct {
	Version   string `json:"version"`
	StartedAt string `json:"started_at"`
	URL       string `json:"url"`
}

func main() {
	startedAt := time.Now().UTC().Format(time.RFC3339)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen failed: %v\n", err)
		os.Exit(1)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	if err := writeServerInfo(serverInfo{
		URL:       baseURL,
		Port:      port,
		PID:       os.Getpid(),
		StartedAt: startedAt,
		Version:   appVersion,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "write runtime metadata failed: %v\n", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, healthResponse{
			Version:   appVersion,
			StartedAt: startedAt,
			URL:       baseURL,
		})
	})

	fmt.Printf("NETSEC_SK_URL=%s\n", baseURL)

	if err := http.Serve(ln, mux); err != nil {
		fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
		os.Exit(1)
	}
}

func writeServerInfo(info serverInfo) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	runtimeDir := filepath.Join(home, ".netsec-sk", "runtime")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(runtimeDir, "server.json")
	payload, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o644)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
