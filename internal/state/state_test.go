package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHashCanonicalization(t *testing.T) {
	s1 := map[string]any{
		"snapshot_version": 1,
		"source": map[string]any{
			"tsf_id":             "S1|a.tgz",
			"tsf_original_name":  "a.tgz",
			"input_archive_name": "in-a.tgz",
			"ingested_at_utc":    "2026-02-09T00:00:00Z",
		},
		"state_sha256": "deadbeef",
		"device": map[string]any{
			"id": "S1",
		},
		"licenses": []any{
			map[string]any{"name": "Threat", "status": "ok"},
			map[string]any{"name": "DNS", "status": "ok"},
		},
		"network": map[string]any{
			"interfaces": []any{
				map[string]any{"name": "ethernet1/2"},
				map[string]any{"name": "ethernet1/1"},
			},
			"zones": []any{
				map[string]any{"name": "trust"},
				map[string]any{"name": "untrust"},
			},
		},
		"routing": map[string]any{
			"virtual_routers": []any{
				map[string]any{"name": "vr2"},
				map[string]any{"name": "vr1"},
			},
		},
	}
	s2 := map[string]any{
		"snapshot_version": 1,
		"source": map[string]any{
			"tsf_id":             "S1|renamed.tgz",
			"tsf_original_name":  "renamed.tgz",
			"input_archive_name": "in-b.tgz",
			"ingested_at_utc":    "2026-02-09T01:00:00Z",
		},
		"state_sha256": "cafebabe",
		"device": map[string]any{
			"id": "S1",
		},
		"licenses": []any{
			map[string]any{"name": "DNS", "status": "ok"},
			map[string]any{"name": "Threat", "status": "ok"},
		},
		"network": map[string]any{
			"interfaces": []any{
				map[string]any{"name": "ethernet1/1"},
				map[string]any{"name": "ethernet1/2"},
			},
			"zones": []any{
				map[string]any{"name": "untrust"},
				map[string]any{"name": "trust"},
			},
		},
		"routing": map[string]any{
			"virtual_routers": []any{
				map[string]any{"name": "vr1"},
				map[string]any{"name": "vr2"},
			},
		},
	}

	h1, err := ComputeStateSHA256(s1)
	if err != nil {
		t.Fatalf("ComputeStateSHA256(s1) err = %v", err)
	}
	h2, err := ComputeStateSHA256(s2)
	if err != nil {
		t.Fatalf("ComputeStateSHA256(s2) err = %v", err)
	}

	if h1 != h2 {
		t.Fatalf("hash mismatch for same logical state: %s != %s", h1, h2)
	}
}

func TestUnchangedStateSkip(t *testing.T) {
	root := t.TempDir()
	latestPath := filepath.Join(root, "latest.json")

	oldSnapshot := map[string]any{
		"snapshot_version": 1,
		"source": map[string]any{
			"tsf_id":             "old",
			"tsf_original_name":  "old.tgz",
			"input_archive_name": "old.tgz",
			"ingested_at_utc":    "2026-02-09T00:00:00Z",
		},
		"state_sha256": "old-hash",
		"device": map[string]any{
			"id": "S1",
		},
	}
	encoded, err := json.Marshal(oldSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(latestPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}

	newSnapshotSame := map[string]any{
		"snapshot_version": 1,
		"source": map[string]any{
			"tsf_id":             "new",
			"tsf_original_name":  "new.tgz",
			"input_archive_name": "new.tgz",
			"ingested_at_utc":    "2026-02-09T02:00:00Z",
		},
		"state_sha256": "new-hash",
		"device": map[string]any{
			"id": "S1",
		},
	}
	unchanged, _, err := IsStateUnchanged(newSnapshotSame, latestPath)
	if err != nil {
		t.Fatalf("IsStateUnchanged(same) err = %v", err)
	}
	if !unchanged {
		t.Fatal("expected unchanged state")
	}

	newSnapshotChanged := map[string]any{
		"snapshot_version": 1,
		"source": map[string]any{
			"tsf_id":             "new",
			"tsf_original_name":  "new.tgz",
			"input_archive_name": "new.tgz",
			"ingested_at_utc":    "2026-02-09T03:00:00Z",
		},
		"state_sha256": "another-hash",
		"device": map[string]any{
			"id": "S2",
		},
	}
	unchanged, _, err = IsStateUnchanged(newSnapshotChanged, latestPath)
	if err != nil {
		t.Fatalf("IsStateUnchanged(changed) err = %v", err)
	}
	if unchanged {
		t.Fatal("expected changed state")
	}
}
