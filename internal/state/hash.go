package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

func ComputeStateSHA256(snapshot map[string]any) (string, error) {
	normalized := normalizeMap(snapshot, "")
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func normalizeMap(in map[string]any, parentKey string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		if parentKey == "" && (k == "source" || k == "state_sha256") {
			continue
		}
		out[k] = normalizeValue(v, k)
	}
	return out
}

func normalizeValue(v any, parentKey string) any {
	switch t := v.(type) {
	case map[string]any:
		return normalizeMap(t, parentKey)
	case []any:
		items := make([]any, 0, len(t))
		for _, it := range t {
			items = append(items, normalizeValue(it, parentKey))
		}
		sortArray(items, parentKey)
		return items
	default:
		return t
	}
}

func sortArray(items []any, parentKey string) {
	if len(items) == 0 {
		return
	}

	if parentKey == "members_serials" || parentKey == "templates" {
		strs := make([]string, 0, len(items))
		for _, it := range items {
			s, ok := it.(string)
			if !ok {
				return
			}
			strs = append(strs, s)
		}
		sort.Strings(strs)
		for i := range strs {
			items[i] = strs[i]
		}
		return
	}

	allNamed := true
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			allNamed = false
			break
		}
		if _, ok := m["name"].(string); !ok {
			allNamed = false
			break
		}
	}
	if !allNamed {
		return
	}

	sort.SliceStable(items, func(i, j int) bool {
		mi := items[i].(map[string]any)
		mj := items[j].(map[string]any)
		return mi["name"].(string) < mj["name"].(string)
	})
}
