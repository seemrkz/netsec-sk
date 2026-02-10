package state

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// ChangedJSONPaths returns normalized dot/index paths for logical state deltas.
// Volatile top-level keys source/state_sha256 are ignored.
func ChangedJSONPaths(previous, current map[string]any) []string {
	if previous == nil {
		previous = map[string]any{}
	}
	if current == nil {
		current = map[string]any{}
	}

	changed := map[string]struct{}{}
	diffMap(previous, current, "", changed)

	out := make([]string, 0, len(changed))
	for p := range changed {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func ChangedScope(previous, current map[string]any) string {
	paths := ChangedJSONPaths(previous, current)
	if len(paths) == 0 {
		return "other"
	}

	buckets := map[string]struct{}{}
	for _, p := range paths {
		buckets[scopeBucket(p)] = struct{}{}
	}

	order := []string{"device", "feature", "route", "other"}
	out := make([]string, 0, len(order))
	for _, name := range order {
		if _, ok := buckets[name]; ok {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return "other"
	}
	return strings.Join(out, ",")
}

func BuildChangedStatePaths(envID, entityType, entityID, snapshotFile string) []string {
	entityDir := "devices"
	if entityType == "panorama" {
		entityDir = "panorama"
	}

	paths := []string{
		filepath.ToSlash(filepath.Join("envs", envID, "state", "commits.ndjson")),
		filepath.ToSlash(filepath.Join("envs", envID, "state", entityDir, entityID, "latest.json")),
		filepath.ToSlash(filepath.Join("envs", envID, "state", entityDir, entityID, "snapshots", snapshotFile)),
	}
	sort.Strings(paths)
	return paths
}

func scopeBucket(path string) string {
	root := path
	if idx := strings.Index(root, "."); idx >= 0 {
		root = root[:idx]
	}
	if idx := strings.Index(root, "["); idx >= 0 {
		root = root[:idx]
	}

	switch root {
	case "device", "panorama_instance":
		return "device"
	case "ha", "network", "licenses", "panorama_ha", "panorama_config":
		return "feature"
	case "routing":
		return "route"
	default:
		return "other"
	}
}

func diffMap(prev, cur map[string]any, prefix string, out map[string]struct{}) {
	keys := map[string]struct{}{}
	for k := range prev {
		keys[k] = struct{}{}
	}
	for k := range cur {
		keys[k] = struct{}{}
	}

	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	for _, k := range sorted {
		if prefix == "" && (k == "source" || k == "state_sha256") {
			continue
		}
		next := k
		if prefix != "" {
			next = prefix + "." + k
		}
		pv, pok := prev[k]
		cv, cok := cur[k]
		if !pok || !cok {
			out[next] = struct{}{}
			continue
		}
		diffValue(pv, cv, next, out)
	}
}

func diffValue(prev, cur any, prefix string, out map[string]struct{}) {
	switch p := prev.(type) {
	case map[string]any:
		c, ok := cur.(map[string]any)
		if !ok {
			out[prefix] = struct{}{}
			return
		}
		diffMap(p, c, prefix, out)
	case []any:
		c, ok := cur.([]any)
		if !ok {
			out[prefix] = struct{}{}
			return
		}
		if len(p) != len(c) {
			out[prefix] = struct{}{}
		}
		n := len(p)
		if len(c) < n {
			n = len(c)
		}
		for i := 0; i < n; i++ {
			diffValue(p[i], c[i], fmt.Sprintf("%s[%d]", prefix, i), out)
		}
	default:
		if !reflect.DeepEqual(prev, cur) {
			out[prefix] = struct{}{}
		}
	}
}
