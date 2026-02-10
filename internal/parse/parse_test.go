package parse

import "testing"

func TestFirewallSnapshotRequiredFields(t *testing.T) {
	ctx := ParseContext{TSFID: "S1|fw.tgz", TSFOriginalName: "fw.tgz", InputArchiveName: "x.tgz", IngestedAtUTC: "2026-02-09T00:00:00Z"}
	files := map[string]string{
		"a.txt": "firewall\nserial: S1\nhostname: fw1\nmgmt_ip: 10.0.0.1\nmodel: PA-440\nsw_version: 11.0.0",
	}

	out, partial, err := ParseFirewallSnapshot(ctx, files)
	if err != nil {
		t.Fatalf("ParseFirewallSnapshot() err = %v", err)
	}
	if partial {
		t.Fatal("expected non-partial parse")
	}
	if out["snapshot_version"] != 1 || out["state_sha256"] == "" {
		t.Fatal("missing required envelope fields")
	}
	src := out["source"].(map[string]any)
	if src["tsf_id"] != "S1|fw.tgz" || src["tsf_original_name"] != "fw.tgz" || src["input_archive_name"] != "x.tgz" || src["ingested_at_utc"] == "" {
		t.Fatal("missing required source fields")
	}
	dev := out["device"].(map[string]any)
	if dev["id"] != "S1" || dev["serial"] != "S1" {
		t.Fatal("missing required firewall identity fields")
	}
}

func TestFirewallSnapshotAcceptsCommonTSFKeyVariants(t *testing.T) {
	ctx := ParseContext{TSFID: "S1|techsupport.tgz", TSFOriginalName: "techsupport.tgz", InputArchiveName: "techsupport.tgz", IngestedAtUTC: "2026-02-09T00:00:00Z"}
	files := map[string]string{
		"a.txt": "serial: S1\nhostname: fw1\nip-address: 10.0.0.1\nmodel: PA-5450\nsw-version: 11.2.4-h7\npanorama\n",
	}

	out, err := ParseSnapshot(ctx, files)
	if err != nil {
		t.Fatalf("ParseSnapshot() err=%v", err)
	}
	if out.Result != "ok" {
		t.Fatalf("result=%q, want ok", out.Result)
	}
	if out.EntityType != EntityFirewall {
		t.Fatalf("entityType=%q, want firewall", out.EntityType)
	}
	dev := out.Snapshot["device"].(map[string]any)
	if dev["mgmt_ip"] != "10.0.0.1" || dev["sw_version"] != "11.2.4-h7" {
		t.Fatalf("unexpected device fields: %#v", dev)
	}
}

func TestPanoramaSnapshotRequiredFields(t *testing.T) {
	ctx := ParseContext{TSFID: "P1|p.tgz", TSFOriginalName: "p.tgz", InputArchiveName: "p.tgz", IngestedAtUTC: "2026-02-09T00:00:00Z"}
	files := map[string]string{
		"a.txt": "panorama\nserial: P1\nhostname: p1\nmgmt_ip: 10.0.0.2\nversion: 11.0.0",
	}

	out, partial, err := ParsePanoramaSnapshot(ctx, files)
	if err != nil {
		t.Fatalf("ParsePanoramaSnapshot() err = %v", err)
	}
	if partial {
		t.Fatal("expected non-partial parse")
	}
	if out["snapshot_version"] != 1 || out["state_sha256"] == "" {
		t.Fatal("missing required envelope fields")
	}
	inst := out["panorama_instance"].(map[string]any)
	if inst["id"] != "P1" || inst["serial"] != "P1" {
		t.Fatal("missing required panorama identity fields")
	}
	cfg := out["panorama_config"].(map[string]any)
	if cfg["device_groups"] == nil || cfg["templates"] == nil || cfg["template_stacks"] == nil || cfg["managed_devices"] == nil {
		t.Fatal("missing required panorama config fields")
	}
}

func TestParseErrorClassification(t *testing.T) {
	ctx := ParseContext{TSFID: "x", TSFOriginalName: "x", InputArchiveName: "x", IngestedAtUTC: "2026-02-09T00:00:00Z"}

	_, err := ParseSnapshot(ctx, map[string]string{"a.txt": "unclassified\nserial: X"})
	if err == nil {
		t.Fatal("expected fatal when entity type cannot be classified")
	}

	_, err = ParseSnapshot(ctx, map[string]string{"a.txt": "firewall\nhostname: fw"})
	if err == nil {
		t.Fatal("expected fatal when entity id cannot be derived")
	}

	out, err := ParseSnapshot(ctx, map[string]string{"a.txt": "firewall\nserial: F1"})
	if err != nil {
		t.Fatalf("unexpected error for partial parse: %v", err)
	}
	if out.Result != "parse_error_partial" {
		t.Fatalf("result = %q, want parse_error_partial", out.Result)
	}
}

func TestPrototypeMinimumFields(t *testing.T) {
	ctx := ParseContext{TSFID: "S1|fw.tgz", TSFOriginalName: "fw.tgz", InputArchiveName: "fw.tgz", IngestedAtUTC: "2026-02-09T00:00:00Z"}

	_, err := ParseSnapshot(ctx, map[string]string{"a.txt": "firewall\nhostname: fw1\nmgmt_ip: 10.0.0.1"})
	if err == nil {
		t.Fatal("expected fatal when minimum firewall identity fields are missing")
	}

	out, err := ParseSnapshot(ctx, map[string]string{"a.txt": "firewall\nserial: F2"})
	if err != nil {
		t.Fatalf("unexpected error for partial parse: %v", err)
	}
	if out.Result != "parse_error_partial" {
		t.Fatalf("result = %q, want parse_error_partial", out.Result)
	}
	if out.Snapshot["snapshot_version"] != 1 || out.Snapshot["source"] == nil || out.Snapshot["state_sha256"] == nil {
		t.Fatalf("partial parse missing required snapshot envelope: %#v", out.Snapshot)
	}
	device := out.Snapshot["device"].(map[string]any)
	if device["id"] != "F2" || device["serial"] != "F2" {
		t.Fatalf("partial parse missing required identity fields: %#v", device)
	}
}
