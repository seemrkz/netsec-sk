package state

import (
	"reflect"
	"testing"
)

func TestChangedScopeClassifier(t *testing.T) {
	previous := map[string]any{
		"device": map[string]any{
			"id":       "S1",
			"hostname": "fw-old",
		},
		"network": map[string]any{
			"interfaces": []any{
				map[string]any{"name": "ethernet1/1"},
			},
		},
		"routing": map[string]any{
			"virtual_routers": []any{
				map[string]any{"name": "default"},
			},
		},
		"source": map[string]any{"tsf_id": "old"},
	}
	current := map[string]any{
		"device": map[string]any{
			"id":       "S1",
			"hostname": "fw-new",
		},
		"network": map[string]any{
			"interfaces": []any{
				map[string]any{"name": "ethernet1/1"},
				map[string]any{"name": "ethernet1/2"},
			},
		},
		"routing": map[string]any{
			"virtual_routers": []any{
				map[string]any{"name": "default"},
				map[string]any{"name": "vr2"},
			},
		},
		"source": map[string]any{"tsf_id": "new"},
	}

	got := ChangedScope(previous, current)
	if got != "device,feature,route" {
		t.Fatalf("ChangedScope() = %q, want %q", got, "device,feature,route")
	}
}

func TestChangedScopeIncludesRouteWhenRoutingChanges(t *testing.T) {
	previous := map[string]any{
		"routing": map[string]any{
			"virtual_routers": []any{
				map[string]any{"name": "default", "counts": map[string]any{"static_routes_configured_v4": float64(1)}},
			},
		},
	}
	current := map[string]any{
		"routing": map[string]any{
			"virtual_routers": []any{
				map[string]any{"name": "default", "counts": map[string]any{"static_routes_configured_v4": float64(2)}},
			},
		},
	}

	got := ChangedScope(previous, current)
	if got != "route" {
		t.Fatalf("ChangedScope() = %q, want %q", got, "route")
	}
}

func TestChangedScopeIgnoresVolatileEnvelope(t *testing.T) {
	previous := map[string]any{
		"source":      map[string]any{"tsf_id": "old"},
		"state_sha256": "abc",
	}
	current := map[string]any{
		"source":      map[string]any{"tsf_id": "new"},
		"state_sha256": "def",
	}

	paths := ChangedJSONPaths(previous, current)
	if len(paths) != 0 {
		t.Fatalf("ChangedJSONPaths() = %#v, want empty", paths)
	}
}

func TestBuildChangedStatePathsLexicalRepoRelative(t *testing.T) {
	got := BuildChangedStatePaths("prod", "firewall", "S1", "20260210_hash.json")
	want := []string{
		"envs/prod/state/commits.ndjson",
		"envs/prod/state/devices/S1/latest.json",
		"envs/prod/state/devices/S1/snapshots/20260210_hash.json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildChangedStatePaths() = %#v, want %#v", got, want)
	}
}
