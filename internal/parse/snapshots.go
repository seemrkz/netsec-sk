package parse

import "strings"

func ParseFirewallSnapshot(ctx ParseContext, files map[string]string) (map[string]any, bool, error) {
	serial := firstSerial(files)
	if serial == "" {
		return nil, false, ErrParseFatal
	}

	content := strings.ToLower(joinContent(files))
	partial := !strings.Contains(content, "hostname:") || !strings.Contains(content, "mgmt_ip:")
	device := map[string]any{
		"id":         serial,
		"hostname":   getKV(files, "hostname"),
		"serial":     serial,
		"model":      getKV(files, "model"),
		"sw_version": getKV(files, "sw_version"),
		"mgmt_ip":    getKV(files, "mgmt_ip"),
	}

	snapshot := baseSnapshot(ctx)
	snapshot["device"] = device
	snapshot["ha"] = map[string]any{"enabled": false, "mode": "unknown", "local_state": "", "peer_serial": ""}
	snapshot["licenses"] = []any{}
	snapshot["network"] = map[string]any{"interfaces": []any{}, "zones": []any{}}
	snapshot["routing"] = map[string]any{"virtual_routers": []any{}}
	return snapshot, partial, nil
}

func ParsePanoramaSnapshot(ctx ParseContext, files map[string]string) (map[string]any, bool, error) {
	serial := firstSerial(files)
	if serial == "" {
		return nil, false, ErrParseFatal
	}

	content := strings.ToLower(joinContent(files))
	partial := !strings.Contains(content, "hostname:") || !strings.Contains(content, "mgmt_ip:")
	inst := map[string]any{
		"id":       serial,
		"hostname": getKV(files, "hostname"),
		"serial":   serial,
		"model":    getKV(files, "model"),
		"version":  getKV(files, "version"),
		"mgmt_ip":  getKV(files, "mgmt_ip"),
	}
	cfg := map[string]any{
		"device_groups":   []any{},
		"templates":       []any{},
		"template_stacks": []any{},
		"managed_devices": []any{},
	}

	snapshot := baseSnapshot(ctx)
	snapshot["panorama_instance"] = inst
	snapshot["panorama_config"] = cfg
	return snapshot, partial, nil
}

func baseSnapshot(ctx ParseContext) map[string]any {
	return map[string]any{
		"snapshot_version": 1,
		"source": map[string]any{
			"tsf_id":             ctx.TSFID,
			"tsf_original_name":  ctx.TSFOriginalName,
			"input_archive_name": ctx.InputArchiveName,
			"ingested_at_utc":    ctx.IngestedAtUTC,
		},
		"state_sha256": "0000000000000000000000000000000000000000000000000000000000000000",
	}
}

func getKV(files map[string]string, key string) string {
	target := strings.ToLower(key) + ":"
	for _, p := range sortedKeys(files) {
		lines := strings.Split(files[p], "\n")
		for _, line := range lines {
			l := strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(l), target) {
				return strings.TrimSpace(l[len(target):])
			}
		}
	}
	return ""
}
