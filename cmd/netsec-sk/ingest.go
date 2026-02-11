package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"time"
)

type ingestProgress struct {
	Pct     int    `json:"pct"`
	Message string `json:"message"`
}

type ingestStatus struct {
	IngestID    string                 `json:"ingest_id"`
	EnvID       string                 `json:"env_id"`
	Status      string                 `json:"status"`
	Stage       string                 `json:"stage"`
	Progress    ingestProgress         `json:"progress"`
	FinalRecord map[string]any         `json:"final_record,omitempty"`
	RMAPrompt   map[string]any         `json:"rma_prompt,omitempty"`
	PendingData map[string]any         `json:"-"`
	StartedAt   time.Time              `json:"-"`
	StageStart  time.Time              `json:"-"`
	Durations   map[string]int64       `json:"-"`
	ArchiveSHA  string                 `json:"-"`
	Filename    string                 `json:"-"`
	Files       map[string]string      `json:"-"`
	Extracted   map[string]any         `json:"-"`
	StatePatch  map[string]interface{} `json:"-"`
}

type createIngestResponse struct {
	IngestID string `json:"ingest_id"`
}

func (a *app) handleCreateIngest(w http.ResponseWriter, r *http.Request, envID string) {
	if _, status := a.resolveEnvironmentPath(envID); status != http.StatusOK {
		if status == http.StatusGone {
			writeError(w, http.StatusNotFound, "ERR_ENV_ALREADY_DELETED", "environment already deleted")
			return
		}
		writeError(w, http.StatusNotFound, "ERR_ENV_NOT_FOUND", "environment not found")
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_ARCHIVE", "invalid multipart upload")
		return
	}

	f, fh, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_ARCHIVE", "file field is required")
		return
	}
	defer f.Close()

	contents, err := io.ReadAll(io.LimitReader(f, 256<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_ARCHIVE", "failed to read upload")
		return
	}

	ingestID := newUUID()
	status := &ingestStatus{
		IngestID:   ingestID,
		EnvID:      envID,
		Status:     "running",
		Stage:      "receive",
		Progress:   ingestProgress{Pct: 0, Message: "received upload"},
		StartedAt:  time.Now().UTC(),
		StageStart: time.Now().UTC(),
		Durations:  map[string]int64{},
		Filename:   filepath.Base(fh.Filename),
		ArchiveSHA: hashBytes(contents),
	}
	a.storeIngest(status)
	a.processIngest(status, contents)
	writeJSON(w, http.StatusAccepted, createIngestResponse{IngestID: ingestID})
}

func (a *app) handleGetIngestStatus(w http.ResponseWriter, ingestID string) {
	st, ok := a.getIngest(ingestID)
	if !ok {
		writeError(w, http.StatusNotFound, "ERR_INGEST_NOT_FOUND", "ingest not found")
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (a *app) handleRmaDecision(w http.ResponseWriter, r *http.Request, ingestID string) {
	// TASK-00009 implementation populates this endpoint.
	writeError(w, http.StatusNotFound, "ERR_INGEST_NOT_FOUND", "ingest not awaiting user input")
}

func (a *app) processIngest(st *ingestStatus, contents []byte) {
	stageDurations := map[string]int64{}
	setStage := func(stage string, pct int, msg string) {
		now := time.Now().UTC()
		stageDurations[st.Stage] += now.Sub(st.StageStart).Milliseconds()
		st.Stage = stage
		st.StageStart = now
		st.Progress = ingestProgress{Pct: pct, Message: msg}
	}

	setStage("scan", 20, "scanning archive")
	files, err := scanTarGz(contents)
	if err != nil {
		st.Status = "completed"
		st.Stage = "persist"
		st.Progress = ingestProgress{Pct: 100, Message: "ingest failed"}
		st.FinalRecord = finalizeRecord(st, stageDurations, "error", map[string]any{
			"stage":   "scan",
			"code":    "ERR_INVALID_ARCHIVE",
			"message": "archive is not a readable gzip/tar",
		})
		a.storeIngest(st)
		return
	}
	st.Files = files

	setStage("identify", 40, "identifying source")
	setStage("extract", 60, "extracting fields")
	setStage("derive", 80, "deriving state")
	setStage("diff", 90, "computing changes")
	setStage("persist", 95, "persisting results")

	st.Status = "completed"
	st.Progress = ingestProgress{Pct: 100, Message: "completed"}
	st.FinalRecord = finalizeRecord(st, stageDurations, "success", nil)
	a.storeIngest(st)
}

func finalizeRecord(st *ingestStatus, stageDurations map[string]int64, status string, ingestErr map[string]any) map[string]any {
	now := time.Now().UTC()
	stageDurations[st.Stage] += now.Sub(st.StageStart).Milliseconds()
	durationTotal := now.Sub(st.StartedAt).Milliseconds()
	durationCompute := durationTotal

	rec := map[string]any{
		"ingest_id":   st.IngestID,
		"env_id":      st.EnvID,
		"started_at":  st.StartedAt.Format(time.RFC3339),
		"finished_at": now.Format(time.RFC3339),
		"status":      status,
		"source": map[string]any{
			"mode":      "file",
			"filenames": []string{st.Filename},
		},
		"fingerprint_sha256": st.ArchiveSHA,
		"device": map[string]any{
			"device_type": "unknown",
			"serial":      "not_found",
			"hostname":    "not_found",
		},
		"duration_ms_total":    durationTotal,
		"duration_ms_compute":  durationCompute,
		"duration_ms_by_stage": stageDurations,
	}
	if ingestErr != nil {
		rec["error"] = ingestErr
	}
	return rec
}

func scanTarGz(contents []byte) (map[string]string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(contents))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	files := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(tr, 2<<20))
		if err != nil {
			return nil, err
		}
		files[hdr.Name] = string(b)
	}
	return files, nil
}

func (a *app) storeIngest(st *ingestStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	cp := *st
	a.ingests[st.IngestID] = &cp
}

func (a *app) getIngest(ingestID string) (ingestStatus, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	st, ok := a.ingests[ingestID]
	if !ok {
		return ingestStatus{}, false
	}
	cp := *st
	return cp, true
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func writeNDJSONLine(path string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	f, err := openAppend(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(raw); err != nil {
		return err
	}
	return f.Sync()
}
