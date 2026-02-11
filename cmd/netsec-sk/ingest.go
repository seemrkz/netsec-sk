package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
	SourceMode  string                 `json:"-"`
}

type createIngestResponse struct {
	IngestID string `json:"ingest_id"`
}

type rmaDecisionRequest struct {
	Decision              string `json:"decision"`
	TargetLogicalDeviceID string `json:"target_logical_device_id"`
}

func (a *app) handleCreateIngest(w http.ResponseWriter, r *http.Request, envID string) {
	envDir, status := a.resolveEnvironmentPath(envID)
	if status != http.StatusOK {
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
	st := &ingestStatus{
		IngestID:   ingestID,
		EnvID:      envID,
		Status:     "running",
		Stage:      "receive",
		Progress:   ingestProgress{Pct: 5, Message: "received upload"},
		StartedAt:  time.Now().UTC(),
		StageStart: time.Now().UTC(),
		Durations:  map[string]int64{},
		Filename:   filepath.Base(fh.Filename),
		ArchiveSHA: hashBytes(contents),
		SourceMode: ingestSourceMode(r),
	}
	a.storeIngest(st)
	a.processIngest(envDir, st, contents, nil)
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
	st, ok := a.getIngest(ingestID)
	if !ok || st.Status != "awaiting_user" {
		writeError(w, http.StatusNotFound, "ERR_INGEST_NOT_FOUND", "ingest not awaiting user input")
		return
	}
	defer r.Body.Close()
	var req rmaDecisionRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid JSON body")
		return
	}
	if req.Decision == "link_replacement" && strings.TrimSpace(req.TargetLogicalDeviceID) == "" {
		writeError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "target_logical_device_id is required for link_replacement")
		return
	}
	if req.Decision != "link_replacement" && req.Decision != "treat_as_new_device" && req.Decision != "canceled" {
		writeError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid decision")
		return
	}
	envDir, status := a.resolveEnvironmentPath(st.EnvID)
	if status != http.StatusOK {
		writeError(w, http.StatusNotFound, "ERR_ENV_NOT_FOUND", "environment not found")
		return
	}
	decision := map[string]any{"decision": req.Decision, "target_logical_device_id": req.TargetLogicalDeviceID}
	a.processIngest(envDir, &st, nil, decision)
	writeJSON(w, http.StatusOK, map[string]any{"ingest_id": st.IngestID, "status": st.Status})
}

func (a *app) processIngest(envDir string, st *ingestStatus, contents []byte, decision map[string]any) {
	stageDurations := map[string]int64{}
	setStage := func(stage string, pct int, msg string) {
		now := time.Now().UTC()
		stageDurations[st.Stage] += now.Sub(st.StageStart).Milliseconds()
		st.Stage = stage
		st.StageStart = now
		st.Progress = ingestProgress{Pct: pct, Message: msg}
	}

	if decision == nil {
		setStage("scan", 15, "scanning archive")
		files, err := scanTarGz(contents)
		if err != nil {
			final := finalizeRecord(st, stageDurations, "error", map[string]any{"stage": "scan", "code": "ERR_INVALID_ARCHIVE", "message": "archive is not a readable gzip/tar"})
			st.Status = "completed"
			st.Stage = "persist"
			st.Progress = ingestProgress{Pct: 100, Message: "completed with error"}
			st.FinalRecord = final
			_ = writeNDJSONLine(filepath.Join(envDir, "ingest.ndjson"), final)
			a.storeIngest(st)
			return
		}
		st.Files = files

		setStage("identify", 35, "identifying device type")
		extracted := extractFields(files)
		st.Extracted = extracted

		setStage("extract", 55, "extracting normalized fields")
		setStage("derive", 70, "deriving candidate state")

		if isDuplicate(envDir, st.ArchiveSHA) {
			final := finalizeRecord(st, stageDurations, "duplicate", nil)
			populateDeviceFromExtracted(final, extracted)
			st.Status = "completed"
			st.Stage = "persist"
			st.Progress = ingestProgress{Pct: 100, Message: "completed (duplicate)"}
			st.FinalRecord = final
			_ = writeNDJSONLine(filepath.Join(envDir, "ingest.ndjson"), final)
			a.storeIngest(st)
			return
		}

		state, err := loadState(envDir)
		if err != nil {
			final := finalizeRecord(st, stageDurations, "error", map[string]any{"stage": "derive", "code": "ERR_PERSIST_FAILED", "message": "failed to load state"})
			populateDeviceFromExtracted(final, extracted)
			st.Status = "completed"
			st.Stage = "persist"
			st.Progress = ingestProgress{Pct: 100, Message: "completed with error"}
			st.FinalRecord = final
			_ = writeNDJSONLine(filepath.Join(envDir, "ingest.ndjson"), final)
			a.storeIngest(st)
			return
		}

		candidates := findRMACandidates(state, extracted)
		if len(candidates) > 0 {
			setStage("awaiting_user", 80, "awaiting RMA decision")
			st.Status = "awaiting_user"
			st.Stage = "awaiting_user"
			st.Progress = ingestProgress{Pct: 80, Message: "awaiting user"}
			st.RMAPrompt = map[string]any{"required": true, "candidates": candidates}
			st.PendingData = map[string]any{
				"extracted": extracted,
				"durations": stageDurations,
			}
			_ = a.writeRuntimeIngest(st, candidates)
			a.storeIngest(st)
			return
		}
		decision = map[string]any{"decision": "treat_as_new_device", "target_logical_device_id": ""}
	}

	// Decision branch (including default new-device path).
	extracted := st.Extracted
	if extracted == nil && st.PendingData != nil {
		extracted, _ = st.PendingData["extracted"].(map[string]any)
	}
	if extracted == nil {
		if fromRuntime := a.readRuntimeIngestExtracted(st.IngestID); fromRuntime != nil {
			extracted = fromRuntime
		}
	}
	if extracted == nil {
		extracted = map[string]any{"device_type": "unknown", "serial": "not_found", "hostname": "not_found", "model": "not_found", "panos_version": "not_found", "mgmt_ip": "not_found", "managed_device_serials": []string{}}
	}

	if decisionValue(decision) == "canceled" {
		final := finalizeRecord(st, stageDurations, "error", map[string]any{
			"stage":   "awaiting_user",
			"code":    "ERR_USER_ABORTED",
			"message": "user canceled RMA decision",
		})
		populateDeviceFromExtracted(final, extracted)
		addRMARecord(final, true, "canceled")
		_ = writeNDJSONLine(filepath.Join(envDir, "ingest.ndjson"), final)
		_ = a.removeRuntimeIngest(st.IngestID)
		st.Status = "completed"
		st.Stage = "persist"
		st.Progress = ingestProgress{Pct: 100, Message: "completed with user abort"}
		st.FinalRecord = final
		st.RMAPrompt = nil
		st.PendingData = nil
		a.storeIngest(st)
		return
	}

	setStage("diff", 85, "computing canonical diff")
	state, err := loadState(envDir)
	if err != nil {
		final := finalizeRecord(st, stageDurations, "error", map[string]any{"stage": "diff", "code": "ERR_PERSIST_FAILED", "message": "failed to load state"})
		populateDeviceFromExtracted(final, extracted)
		addRMARecord(final, true, decisionValue(decision))
		st.Status = "completed"
		st.Stage = "persist"
		st.Progress = ingestProgress{Pct: 100, Message: "completed with error"}
		st.FinalRecord = final
		_ = writeNDJSONLine(filepath.Join(envDir, "ingest.ndjson"), final)
		a.storeIngest(st)
		return
	}

	beforeHash, _ := hashCanonical(state)
	logicalID, newState := applyExtractedState(state, st, extracted, decision)
	a.sortState(newState)
	a.applyTopology(newState)
	afterHash, _ := hashCanonical(newState)

	setStage("persist", 95, "persisting state and logs")
	statusCode := "success"
	if beforeHash == afterHash {
		statusCode = "no_change"
	}
	a.writeIntro(envDir, newState, statusCode, time.Now().UTC().Format(time.RFC3339))
	final := finalizeRecord(st, stageDurations, statusCode, nil)
	populateDeviceFromExtracted(final, extracted)
	if st.PendingData != nil {
		addRMARecord(final, true, decisionValue(decision))
	}

	if statusCode == "success" {
		if err := a.writeStateAtomic(envDir, newState); err != nil {
			final = finalizeRecord(st, stageDurations, "error", map[string]any{"stage": "persist", "code": "ERR_PERSIST_FAILED", "message": "failed to persist state"})
			populateDeviceFromExtracted(final, extracted)
			if st.PendingData != nil {
				addRMARecord(final, true, decisionValue(decision))
			}
		} else {
			commitID := newUUID()
			commit := map[string]any{
				"commit_id":         commitID,
				"env_id":            st.EnvID,
				"ingest_id":         st.IngestID,
				"timestamp":         time.Now().UTC().Format(time.RFC3339),
				"source_summary":    st.Filename,
				"change_summary":    []string{"device inventory updated"},
				"change_paths":      []string{"/devices/logical", "/topology/inferred_adjacencies"},
				"state_hash_before": beforeHash,
				"state_hash_after":  afterHash,
			}
			_ = writeNDJSONLine(filepath.Join(envDir, "commits.ndjson"), commit)
			final["result"] = map[string]any{"commit_id": commitID, "state_hash_after": afterHash}
			_ = logicalID
		}
	}

	_ = writeNDJSONLine(filepath.Join(envDir, "ingest.ndjson"), final)
	_ = a.removeRuntimeIngest(st.IngestID)
	st.Status = "completed"
	st.Stage = "persist"
	st.Progress = ingestProgress{Pct: 100, Message: "completed"}
	st.FinalRecord = final
	st.RMAPrompt = nil
	st.PendingData = nil
	a.storeIngest(st)
}

func finalizeRecord(st *ingestStatus, stageDurations map[string]int64, status string, ingestErr map[string]any) map[string]any {
	now := time.Now().UTC()
	stageDurations[st.Stage] += now.Sub(st.StageStart).Milliseconds()
	durationTotal := now.Sub(st.StartedAt).Milliseconds()
	durationCompute := durationTotal

	rec := map[string]any{
		"ingest_id":            st.IngestID,
		"env_id":               st.EnvID,
		"started_at":           st.StartedAt.Format(time.RFC3339),
		"finished_at":          now.Format(time.RFC3339),
		"status":               status,
		"source":               map[string]any{"mode": "file", "filenames": []string{st.Filename}},
		"fingerprint_sha256":   st.ArchiveSHA,
		"device":               map[string]any{"device_type": "unknown", "serial": "not_found", "hostname": "not_found"},
		"duration_ms_total":    maxInt64(durationTotal, 0),
		"duration_ms_compute":  maxInt64(durationCompute, 0),
		"duration_ms_by_stage": stageDurations,
	}
	if ingestErr != nil {
		rec["error"] = ingestErr
	}
	if st.SourceMode == "batch" {
		rec["source"] = map[string]any{"mode": "batch", "filenames": []string{st.Filename}}
	}
	return rec
}

func maxInt64(v int64, min int64) int64 {
	if v < min {
		return min
	}
	return v
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
	if len(files) == 0 {
		return nil, fmt.Errorf("no files in archive")
	}
	return files, nil
}

func extractFields(files map[string]string) map[string]any {
	var text strings.Builder
	isPanorama := false
	for name, body := range files {
		lower := strings.ToLower(name + "\n" + body)
		if strings.Contains(lower, "panorama") || strings.Contains(lower, "device-group") {
			isPanorama = true
		}
		text.WriteString("\n")
		text.WriteString(body)
	}
	all := text.String()

	serial := firstMatch(all,
		`(?im)\bserial(?: number)?\s*[:=]\s*([A-Za-z0-9._-]+)`,
		`(?im)\bchassis[-_ ]serial\s*[:=]\s*([A-Za-z0-9._-]+)`)
	hostname := firstMatch(all,
		`(?im)\bhostname\s*[:=]\s*([A-Za-z0-9._-]+)`,
		`(?im)^set\s+deviceconfig\s+system\s+hostname\s+([A-Za-z0-9._-]+)`)
	model := firstMatch(all, `(?im)\bmodel\s*[:=]\s*([A-Za-z0-9._-]+)`)
	panos := firstMatch(all,
		`(?im)\b(?:pan-?os|sw[-_ ]?version|version)\s*[:=]\s*([A-Za-z0-9._-]+)`)
	mgmtIP := firstMatch(all, `(?im)\b(?:mgmt|management)[-_ ]?ip\s*[:=]\s*([0-9]{1,3}(?:\.[0-9]{1,3}){3})`)
	if mgmtIP == "not_found" {
		mgmtIP = firstMatch(all, `(?im)\b([0-9]{1,3}(?:\.[0-9]{1,3}){3})\b`)
	}

	managedSerials := findAllMatches(all, `(?im)\bmanaged[_ -]?serial\s*[:=]\s*([A-Za-z0-9._-]+)`)
	sort.Strings(managedSerials)

	deviceType := "firewall"
	if isPanorama {
		deviceType = "panorama"
	}
	if serial == "not_found" && hostname == "not_found" {
		deviceType = "unknown"
	}

	return map[string]any{
		"device_type":            deviceType,
		"serial":                 serial,
		"hostname":               hostname,
		"model":                  model,
		"panos_version":          panos,
		"mgmt_ip":                mgmtIP,
		"managed_device_serials": managedSerials,
	}
}

func firstMatch(s string, patterns ...string) string {
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		m := re.FindStringSubmatch(s)
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			return strings.TrimSpace(m[1])
		}
	}
	return "not_found"
}

func findAllMatches(s, p string) []string {
	re := regexp.MustCompile(p)
	found := map[string]struct{}{}
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			found[strings.TrimSpace(m[1])] = struct{}{}
		}
	}
	out := make([]string, 0, len(found))
	for v := range found {
		out = append(out, v)
	}
	return out
}

func populateDeviceFromExtracted(record map[string]any, extracted map[string]any) {
	record["device"] = map[string]any{
		"device_type": valueString(extracted["device_type"], "unknown"),
		"serial":      valueString(extracted["serial"], "not_found"),
		"hostname":    valueString(extracted["hostname"], "not_found"),
	}
}

func addRMARecord(record map[string]any, prompted bool, decision string) {
	record["rma"] = map[string]any{"prompted": prompted, "decision": decision}
}

func decisionValue(decision map[string]any) string {
	if decision == nil {
		return ""
	}
	return valueString(decision["decision"], "")
}

func valueString(v any, fallback string) string {
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func loadState(envDir string) (map[string]any, error) {
	path := filepath.Join(envDir, "state.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{"schema_version": "1.0.0", "generated_at": time.Now().UTC().Format(time.RFC3339), "env": map[string]any{"env_id": filepath.Base(envDir), "name": "unknown"}, "devices": map[string]any{"logical": []map[string]any{}}, "topology": map[string]any{"inferred_adjacencies": []map[string]any{}}}, nil
		}
		return nil, err
	}
	var state map[string]any
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, err
	}
	return state, nil
}

func isDuplicate(envDir, hash string) bool {
	path := filepath.Join(envDir, "ingest.ndjson")
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		var rec map[string]any
		if json.Unmarshal([]byte(line), &rec) != nil {
			continue
		}
		if valueString(rec["fingerprint_sha256"], "") == hash {
			return true
		}
	}
	return false
}

func findRMACandidates(state map[string]any, extracted map[string]any) []map[string]any {
	hostname := valueString(extracted["hostname"], "not_found")
	serial := valueString(extracted["serial"], "not_found")
	if hostname == "not_found" || serial == "not_found" {
		return nil
	}
	logical := logicalDevices(state)
	out := make([]map[string]any, 0)
	for _, d := range logical {
		current, _ := d["current"].(map[string]any)
		identity, _ := current["identity"].(map[string]any)
		h := valueString(identity["hostname"], "not_found")
		s := valueString(identity["serial"], "not_found")
		if h == hostname && s != "not_found" && s != serial {
			out = append(out, map[string]any{
				"logical_device_id": valueString(d["logical_device_id"], ""),
				"current_serial":    s,
				"current_hostname":  h,
			})
		}
	}
	return out
}

func logicalDevices(state map[string]any) []map[string]any {
	devices, _ := state["devices"].(map[string]any)
	if devices == nil {
		devices = map[string]any{}
		state["devices"] = devices
	}
	arr, _ := devices["logical"].([]any)
	out := make([]map[string]any, 0, len(arr))
	for _, it := range arr {
		m, _ := it.(map[string]any)
		if m != nil {
			out = append(out, m)
		}
	}
	return out
}

func applyExtractedState(state map[string]any, st *ingestStatus, extracted map[string]any, decision map[string]any) (string, map[string]any) {
	now := time.Now().UTC().Format(time.RFC3339)

	serial := valueString(extracted["serial"], "not_found")
	deviceType := valueString(extracted["device_type"], "unknown")

	logical := logicalDevices(state)
	targetID := valueString(decision["target_logical_device_id"], "")
	d := valueString(decision["decision"], "treat_as_new_device")

	findByID := func(id string) map[string]any {
		for _, dev := range logical {
			if valueString(dev["logical_device_id"], "") == id {
				return dev
			}
		}
		return nil
	}
	findBySerial := func(s string) map[string]any {
		for _, dev := range logical {
			cur, _ := dev["current"].(map[string]any)
			idn, _ := cur["identity"].(map[string]any)
			if valueString(idn["serial"], "") == s {
				return dev
			}
		}
		return nil
	}

	var target map[string]any
	if d == "link_replacement" && targetID != "" {
		target = findByID(targetID)
	}
	if target == nil {
		target = findBySerial(serial)
	}
	if target == nil && d != "link_replacement" {
		target = map[string]any{
			"logical_device_id": newUUID(),
			"device_type":       mapDeviceType(deviceType),
			"serial_history":    []any{},
		}
		logical = append(logical, target)
	}

	if target == nil {
		// link requested for unknown target; treat as canceled/no change behavior.
		return "", state
	}

	if d != "link_replacement" && target != nil && serial != "not_found" {
		cur, _ := target["current"].(map[string]any)
		idn, _ := cur["identity"].(map[string]any)
		if valueString(target["device_type"], "") == mapDeviceType(deviceType) &&
			valueString(idn["hostname"], "not_found") == valueString(extracted["hostname"], "not_found") &&
			valueString(idn["serial"], "not_found") == valueString(extracted["serial"], "not_found") &&
			valueString(idn["model"], "not_found") == valueString(extracted["model"], "not_found") &&
			valueString(idn["panos_version"], "not_found") == valueString(extracted["panos_version"], "not_found") &&
			valueString(idn["mgmt_ip"], "not_found") == valueString(extracted["mgmt_ip"], "not_found") {
			return valueString(target["logical_device_id"], ""), state
		}
	}

	state["schema_version"] = "1.0.0"
	state["generated_at"] = now
	if _, ok := state["topology"]; !ok {
		state["topology"] = map[string]any{"inferred_adjacencies": []map[string]any{}}
	}

	target["device_type"] = mapDeviceType(deviceType)
	target["current"] = buildCurrentSnapshot(st, extracted)
	target["serial_history"] = updateSerialHistory(target["serial_history"], serial, st.IngestID, now)

	arr := make([]any, 0, len(logical))
	for _, dev := range logical {
		arr = append(arr, dev)
	}
	state["devices"].(map[string]any)["logical"] = arr
	return valueString(target["logical_device_id"], ""), state
}

func mapDeviceType(v string) string {
	if v == "panorama" {
		return "panorama"
	}
	return "firewall"
}

func updateSerialHistory(existing any, serial, ingestID, now string) []any {
	arr, _ := existing.([]any)
	if serial == "not_found" {
		return arr
	}
	for i, it := range arr {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		if valueString(m["serial"], "") == serial {
			m["last_seen_ingest_id"] = ingestID
			m["last_seen_at"] = now
			arr[i] = m
			return arr
		}
	}
	arr = append(arr, map[string]any{
		"serial":               serial,
		"first_seen_ingest_id": ingestID,
		"last_seen_ingest_id":  ingestID,
		"first_seen_at":        now,
		"last_seen_at":         now,
	})
	sort.Slice(arr, func(i, j int) bool {
		mi, _ := arr[i].(map[string]any)
		mj, _ := arr[j].(map[string]any)
		return valueString(mi["serial"], "") < valueString(mj["serial"], "")
	})
	return arr
}

func buildCurrentSnapshot(st *ingestStatus, extracted map[string]any) map[string]any {
	deviceType := valueString(extracted["device_type"], "unknown")
	snapshot := map[string]any{
		"observed_at": time.Now().UTC().Format(time.RFC3339),
		"source": map[string]any{
			"ingest_id":          st.IngestID,
			"fingerprint_sha256": st.ArchiveSHA,
		},
		"identity": map[string]any{
			"hostname":      valueString(extracted["hostname"], "not_found"),
			"model":         valueString(extracted["model"], "not_found"),
			"serial":        valueString(extracted["serial"], "not_found"),
			"panos_version": valueString(extracted["panos_version"], "not_found"),
			"mgmt_ip":       valueString(extracted["mgmt_ip"], "not_found"),
		},
		"management": map[string]any{
			"management_type":  "undetermined",
			"panorama_servers": []string{},
			"cloud_mode":       "not_found",
		},
		"ha": map[string]any{
			"enabled": "unknown",
			"mode":    "not_found",
			"peer":    "not_found",
		},
		"licenses": []any{},
		"cloud_logging_service_forwarding": map[string]any{
			"enabled":                              "unknown",
			"region":                               "not_found",
			"enhanced_application_logging_enabled": "unknown",
			"source_path":                          "not_found",
		},
		"network": map[string]any{
			"interfaces":     []any{},
			"zones":          []any{},
			"routes_config":  []any{},
			"routes_runtime": []any{},
		},
	}
	if deviceType == "panorama" {
		mds, _ := extracted["managed_device_serials"].([]string)
		snapshot["panorama"] = map[string]any{
			"managed_device_serials": mds,
			"device_groups":          []any{},
			"template_stacks":        []any{},
			"templates":              []any{},
		}
	}
	return snapshot
}

func (a *app) sortState(state map[string]any) {
	logical := logicalDevices(state)
	sort.Slice(logical, func(i, j int) bool {
		return valueString(logical[i]["logical_device_id"], "") < valueString(logical[j]["logical_device_id"], "")
	})
	arr := make([]any, 0, len(logical))
	for _, dev := range logical {
		arr = append(arr, dev)
	}
	state["devices"].(map[string]any)["logical"] = arr
}

func (a *app) applyTopology(state map[string]any) {
	logical := logicalDevices(state)
	type route struct {
		id   string
		cidr string
	}
	routes := []route{}
	for _, dev := range logical {
		cur, _ := dev["current"].(map[string]any)
		network, _ := cur["network"].(map[string]any)
		runtime, _ := network["routes_runtime"].([]any)
		config, _ := network["routes_config"].([]any)
		collect := func(items []any) {
			for _, it := range items {
				r, _ := it.(map[string]any)
				dst := valueString(r["destination"], "")
				if dst == "" || dst == "0.0.0.0/0" {
					continue
				}
				routes = append(routes, route{id: valueString(dev["logical_device_id"], ""), cidr: dst})
			}
		}
		collect(runtime)
		collect(config)
	}
	edges := []map[string]any{}
	for i := 0; i < len(routes); i++ {
		for j := i + 1; j < len(routes); j++ {
			if routes[i].id == routes[j].id {
				continue
			}
			overlap := overlapCIDR(routes[i].cidr, routes[j].cidr)
			if overlap == "" {
				continue
			}
			aID, bID := routes[i].id, routes[j].id
			if aID > bID {
				aID, bID = bID, aID
			}
			edges = append(edges, map[string]any{
				"fw_a_logical_device_id": aID,
				"fw_b_logical_device_id": bID,
				"overlap_cidrs":          []string{overlap},
				"evidence": []map[string]any{{
					"fw_a_route": routes[i].cidr,
					"fw_b_route": routes[j].cidr,
					"reason":     "connected",
				}},
			})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		aA := valueString(edges[i]["fw_a_logical_device_id"], "")
		bA := valueString(edges[j]["fw_a_logical_device_id"], "")
		if aA != bA {
			return aA < bA
		}
		return valueString(edges[i]["fw_b_logical_device_id"], "") < valueString(edges[j]["fw_b_logical_device_id"], "")
	})
	state["topology"].(map[string]any)["inferred_adjacencies"] = edges
}

func overlapCIDR(a, b string) string {
	if a == b {
		return a
	}
	// Minimal overlap approximation used in MVP tasks 4-9 implementation.
	if strings.HasSuffix(a, "/32") && strings.HasSuffix(b, "/32") {
		if a == b {
			return a
		}
	}
	return ""
}

func (a *app) writeStateAtomic(envDir string, state map[string]any) error {
	path := filepath.Join(envDir, "state.json")
	bak := filepath.Join(envDir, "state.json.bak")
	tmp := path + ".tmp"
	oldState, hadOld := []byte(nil), false
	if old, err := os.ReadFile(path); err == nil {
		hadOld = true
		oldState = old
		if err := os.WriteFile(bak, old, 0o644); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		if hadOld {
			_ = os.WriteFile(path, oldState, 0o644)
		}
		return err
	}
	return nil
}

func (a *app) writeIntro(envDir string, state map[string]any, lastStatus, finishedAt string) {
	meta, _ := readMeta(filepath.Join(envDir, "meta.json"))
	logical := logicalDevices(state)
	firewalls := 0
	panoramas := 0
	for _, dev := range logical {
		if valueString(dev["device_type"], "") == "panorama" {
			panoramas++
		} else {
			firewalls++
		}
	}
	text := fmt.Sprintf("# %s\n\nThis is a derived environment snapshot generated at %s for deterministic inspection.\n\nQuick facts\n- logical devices: %d\n- firewalls: %d\n- panoramas: %d\n- last ingest status: %s at %s\n\nWhere to look in state.json\n- devices list: /devices/logical\n- inferred adjacencies: /topology/inferred_adjacencies\n- per-device network inventory: /devices/logical[i]/current/network\n\nAI Agent notes\n- This file is a derived snapshot; consult commits.ndjson for history.\n- Ingest attempts are recorded in ingest.ndjson (including duplicates/errors).\n- TSF bytes are not retained; provenance is tracked by ingest IDs and fingerprints.\n", meta.Name, time.Now().UTC().Format(time.RFC3339), len(logical), firewalls, panoramas, lastStatus, finishedAt)
	text = text + "\n"
	_ = writeFileAtomic(filepath.Join(envDir, "intro.md"), []byte(text))
}

func hashCanonical(state map[string]any) (string, error) {
	b, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(append(b, '\n'))
	return hex.EncodeToString(h[:]), nil
}

func (a *app) writeRuntimeIngest(st *ingestStatus, candidates []map[string]any) error {
	path := filepath.Join(a.storage, "runtime", "ingests", st.IngestID+".json")
	payload := map[string]any{
		"ingest_id":  st.IngestID,
		"env_id":     st.EnvID,
		"started_at": st.StartedAt.Format(time.RFC3339),
		"device_identity": map[string]any{
			"hostname": valueString(st.Extracted["hostname"], "not_found"),
			"serial":   valueString(st.Extracted["serial"], "not_found"),
		},
		"extracted_payload": st.Extracted,
		"rma_candidates":    candidates,
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (a *app) removeRuntimeIngest(ingestID string) error {
	path := filepath.Join(a.storage, "runtime", "ingests", ingestID+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (a *app) readRuntimeIngestExtracted(ingestID string) map[string]any {
	path := filepath.Join(a.storage, "runtime", "ingests", ingestID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var payload map[string]any
	if json.Unmarshal(b, &payload) != nil {
		return nil
	}
	ex, _ := payload["extracted_payload"].(map[string]any)
	return ex
}

func (a *app) cleanupRuntimeIngestsTTL() {
	dir := filepath.Join(a.storage, "runtime", "ingests")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(p)
		}
	}
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

func ingestSourceMode(r *http.Request) string {
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("mode")), "batch") {
		return "batch"
	}
	return "file"
}

func writeFileAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
