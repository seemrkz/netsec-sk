package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
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

type app struct {
	startedAt string
	baseURL   string
	storage   string
	mu        sync.RWMutex
	ingests   map[string]*ingestStatus
}

type envMeta struct {
	EnvID         string `json:"env_id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	SoftDeleted   bool   `json:"soft_deleted"`
	SoftDeletedAt string `json:"soft_deleted_at"`
}

type createEnvRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type listEnvsResponse struct {
	Environments []envMeta `json:"environments"`
}

type deleteEnvResponse struct {
	EnvID         string `json:"env_id"`
	SoftDeleted   bool   `json:"soft_deleted"`
	SoftDeletedAt string `json:"soft_deleted_at"`
}

type envStateResponse struct {
	State any `json:"state"`
}

type commitsResponse struct {
	EnvID   string           `json:"env_id"`
	Commits []map[string]any `json:"commits"`
}

type errorResponse struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
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
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve home dir failed: %v\n", err)
		os.Exit(1)
	}
	storageRoot := filepath.Join(home, ".netsec-sk")

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

	a := &app{
		startedAt: startedAt,
		baseURL:   baseURL,
		storage:   storageRoot,
		ingests:   map[string]*ingestStatus{},
	}
	a.cleanupRuntimeIngestsTTL()
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.route)

	fmt.Printf("NETSEC_SK_URL=%s\n", baseURL)

	if err := http.Serve(ln, mux); err != nil {
		fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
		os.Exit(1)
	}
}

func (a *app) route(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/health":
		a.handleHealth(w, r)
	case r.URL.Path == "/api/environments":
		a.handleEnvironments(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/ingests/"):
		a.handleIngestByID(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/environments/"):
		a.handleEnvironmentByID(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{
		Version:   appVersion,
		StartedAt: a.startedAt,
		URL:       a.baseURL,
	})
}

func (a *app) handleEnvironments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleListEnvironments(w, r)
	case http.MethodPost:
		a.handleCreateEnvironment(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *app) handleEnvironmentByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/environments/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		a.handleDeleteEnvironment(w, parts[0])
		return
	}

	if len(parts) == 2 && parts[1] == "ingests" {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		a.handleCreateIngest(w, r, parts[0])
		return
	}

	if len(parts) != 2 || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	switch parts[1] {
	case "state":
		a.handleGetEnvironmentState(w, parts[0])
	case "commits":
		a.handleGetEnvironmentCommits(w, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func (a *app) handleIngestByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/ingests/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 1 && r.Method == http.MethodGet {
		a.handleGetIngestStatus(w, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "rma-decision" && r.Method == http.MethodPost {
		a.handleRmaDecision(w, r, parts[0])
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (a *app) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid request body")
		return
	}
	var req createEnvRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "ERR_ENV_NAME_REQUIRED", "environment name is required")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	meta := envMeta{
		EnvID:         newUUID(),
		Name:          req.Name,
		Description:   req.Description,
		CreatedAt:     now,
		UpdatedAt:     now,
		SoftDeleted:   false,
		SoftDeletedAt: "",
	}
	envDir := filepath.Join(a.storage, "environments", meta.EnvID)
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_PERSIST_FAILED", "failed to create environment directory")
		return
	}
	if err := writeMeta(filepath.Join(envDir, "meta.json"), meta); err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_PERSIST_FAILED", "failed to write environment metadata")
		return
	}
	writeJSON(w, http.StatusCreated, meta)
}

func (a *app) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	envsRoot := filepath.Join(a.storage, "environments")
	if err := os.MkdirAll(envsRoot, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_PERSIST_FAILED", "failed to initialize environment store")
		return
	}

	entries, err := os.ReadDir(envsRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_PERSIST_FAILED", "failed to read environments")
		return
	}

	envs := make([]envMeta, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(envsRoot, entry.Name(), "meta.json")
		meta, ok := readMeta(metaPath)
		if !ok || meta.SoftDeleted {
			continue
		}
		envs = append(envs, meta)
	}
	sort.Slice(envs, func(i, j int) bool { return envs[i].EnvID < envs[j].EnvID })
	writeJSON(w, http.StatusOK, listEnvsResponse{Environments: envs})
}

func (a *app) handleDeleteEnvironment(w http.ResponseWriter, envID string) {
	envDir := filepath.Join(a.storage, "environments", envID)
	metaPath := filepath.Join(envDir, "meta.json")
	meta, ok := readMeta(metaPath)
	if !ok {
		if _, err := os.Stat(filepath.Join(a.storage, "trash", envID)); err == nil {
			writeError(w, http.StatusNotFound, "ERR_ENV_ALREADY_DELETED", "environment already deleted")
			return
		}
		writeError(w, http.StatusNotFound, "ERR_ENV_NOT_FOUND", "environment not found")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	meta.SoftDeleted = true
	meta.SoftDeletedAt = now
	meta.UpdatedAt = now
	if err := writeMeta(metaPath, meta); err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_PERSIST_FAILED", "failed to update environment metadata")
		return
	}

	trashDir := filepath.Join(a.storage, "trash")
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_PERSIST_FAILED", "failed to initialize trash directory")
		return
	}
	if err := os.Rename(envDir, filepath.Join(trashDir, envID)); err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_PERSIST_FAILED", "failed to move environment to trash")
		return
	}
	writeJSON(w, http.StatusOK, deleteEnvResponse{
		EnvID:         envID,
		SoftDeleted:   true,
		SoftDeletedAt: now,
	})
}

func writeMeta(path string, meta envMeta) error {
	payload, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o644)
}

func readMeta(path string) (envMeta, bool) {
	var meta envMeta
	payload, err := os.ReadFile(path)
	if err != nil {
		return meta, false
	}
	if err := json.Unmarshal(payload, &meta); err != nil {
		return meta, false
	}
	return meta, true
}

func (a *app) handleGetEnvironmentState(w http.ResponseWriter, envID string) {
	envDir, status := a.resolveEnvironmentPath(envID)
	if status != http.StatusOK {
		if status == http.StatusGone {
			writeError(w, http.StatusNotFound, "ERR_ENV_ALREADY_DELETED", "environment already deleted")
			return
		}
		writeError(w, http.StatusNotFound, "ERR_ENV_NOT_FOUND", "environment not found")
		return
	}

	statePath := filepath.Join(envDir, "state.json")
	payload, err := os.ReadFile(statePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "ERR_ENV_STATE_NOT_FOUND", "environment state not found")
		return
	}

	var state any
	if err := json.Unmarshal(payload, &state); err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_PERSIST_FAILED", "state file is invalid")
		return
	}
	writeJSON(w, http.StatusOK, envStateResponse{State: state})
}

func (a *app) handleGetEnvironmentCommits(w http.ResponseWriter, envID string) {
	envDir, status := a.resolveEnvironmentPath(envID)
	if status != http.StatusOK {
		if status == http.StatusGone {
			writeError(w, http.StatusNotFound, "ERR_ENV_ALREADY_DELETED", "environment already deleted")
			return
		}
		writeError(w, http.StatusNotFound, "ERR_ENV_NOT_FOUND", "environment not found")
		return
	}

	commitsPath := filepath.Join(envDir, "commits.ndjson")
	commits, err := readCommits(commitsPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_PERSIST_FAILED", "failed to read commits")
		return
	}

	sort.Slice(commits, func(i, j int) bool {
		it := stringValue(commits[i]["timestamp"])
		jt := stringValue(commits[j]["timestamp"])
		if it != jt {
			return it > jt
		}
		return stringValue(commits[i]["commit_id"]) < stringValue(commits[j]["commit_id"])
	})

	writeJSON(w, http.StatusOK, commitsResponse{
		EnvID:   envID,
		Commits: commits,
	})
}

func (a *app) resolveEnvironmentPath(envID string) (string, int) {
	envDir := filepath.Join(a.storage, "environments", envID)
	if _, err := os.Stat(envDir); err == nil {
		return envDir, http.StatusOK
	}
	if _, err := os.Stat(filepath.Join(a.storage, "trash", envID)); err == nil {
		return "", http.StatusGone
	}
	return "", http.StatusNotFound
}

func readCommits(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	defer file.Close()

	out := make([]map[string]any, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var commit map[string]any
		if err := json.Unmarshal([]byte(line), &commit); err != nil {
			return nil, err
		}
		out = append(out, commit)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	if len(msg) > 512 {
		msg = msg[:512]
	}
	writeJSON(w, status, errorResponse{
		Code:    code,
		Message: msg,
		Details: map[string]any{},
	})
}

func newUUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		uint32(buf[0])<<24|uint32(buf[1])<<16|uint32(buf[2])<<8|uint32(buf[3]),
		uint16(buf[4])<<8|uint16(buf[5]),
		uint16(buf[6])<<8|uint16(buf[7]),
		uint16(buf[8])<<8|uint16(buf[9]),
		uint64(buf[10])<<40|uint64(buf[11])<<32|uint64(buf[12])<<24|uint64(buf[13])<<16|uint64(buf[14])<<8|uint64(buf[15]),
	)
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

func openAppend(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}
