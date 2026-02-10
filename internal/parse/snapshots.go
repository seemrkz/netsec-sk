package parse

import "strings"

func ParseFirewallSnapshot(ctx ParseContext, files map[string]string) (map[string]any, bool, error) {
	serial := firstSerial(files)
	if serial == "" {
		return nil, false, ErrParseFatal
	}

	hostname := getKVAny(files, "hostname")
	mgmtIP := getKVAny(files, "mgmt_ip", "mgmt-ip", "mgmt ip", "ip-address", "ip address")
	model := getKVAny(files, "model")
	swVersion := getKVAny(files, "sw_version", "sw-version", "sw version", "version")

	partial := hostname == "" || mgmtIP == ""
	device := map[string]any{
		"id":         serial,
		"hostname":   hostname,
		"serial":     serial,
		"model":      model,
		"sw_version": swVersion,
		"mgmt_ip":    mgmtIP,
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

	hostname := getKVAny(files, "hostname")
	mgmtIP := getKVAny(files, "mgmt_ip", "mgmt-ip", "mgmt ip", "ip-address", "ip address")
	model := getKVAny(files, "model")
	version := getKVAny(files, "version", "sw_version", "sw-version", "sw version")

	partial := hostname == "" || mgmtIP == ""
	inst := map[string]any{
		"id":       serial,
		"hostname": hostname,
		"serial":   serial,
		"model":    model,
		"version":  version,
		"mgmt_ip":  mgmtIP,
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

func getKVAny(files map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := getKV(files, key); v != "" {
			return v
		}
	}
	return ""
}
