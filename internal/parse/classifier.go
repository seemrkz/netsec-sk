package parse

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type EntityType string

const (
	EntityFirewall EntityType = "firewall"
	EntityPanorama EntityType = "panorama"
)

var (
	ErrParseFatal = errors.New("parse_error_fatal")
	reSerial      = regexp.MustCompile(`(?im)^\s*(?:serial|serial number|device serial)\s*:\s*(\S+)`)
	reModelLine   = regexp.MustCompile(`(?im)^\s*model\s*:\s*(\S+)`)
)

type ParseContext struct {
	TSFID            string
	TSFOriginalName  string
	InputArchiveName string
	IngestedAtUTC    string
}

type ParseOutput struct {
	EntityType EntityType
	EntityID   string
	Snapshot   map[string]any
	Result     string
}

func ParseSnapshot(ctx ParseContext, files map[string]string) (ParseOutput, error) {
	entityType, err := ClassifyEntity(files)
	if err != nil {
		return ParseOutput{Result: "parse_error_fatal"}, fmt.Errorf("classify entity: %w", err)
	}

	switch entityType {
	case EntityFirewall:
		out, partial, err := ParseFirewallSnapshot(ctx, files)
		if err != nil {
			return ParseOutput{Result: "parse_error_fatal"}, fmt.Errorf("parse firewall snapshot: %w", err)
		}
		result := "ok"
		if partial {
			result = "parse_error_partial"
		}
		return ParseOutput{EntityType: entityType, EntityID: out["device"].(map[string]any)["id"].(string), Snapshot: out, Result: result}, nil
	case EntityPanorama:
		out, partial, err := ParsePanoramaSnapshot(ctx, files)
		if err != nil {
			return ParseOutput{Result: "parse_error_fatal"}, fmt.Errorf("parse panorama snapshot: %w", err)
		}
		result := "ok"
		if partial {
			result = "parse_error_partial"
		}
		return ParseOutput{EntityType: entityType, EntityID: out["panorama_instance"].(map[string]any)["id"].(string), Snapshot: out, Result: result}, nil
	default:
		return ParseOutput{Result: "parse_error_fatal"}, ErrParseFatal
	}
}

func ClassifyEntity(files map[string]string) (EntityType, error) {
	joined := strings.ToLower(joinContent(files))
	// Prefer a model-based classifier when possible. "panorama" may appear in many firewall
	// techsupport dumps (e.g. admin usernames, config paths), so a pure substring match is
	// too noisy for real-world archives.
	if m := reModelLine.FindStringSubmatch(joined); len(m) == 2 {
		model := strings.ToLower(strings.TrimSpace(m[1]))
		switch {
		case strings.HasPrefix(model, "pa-") || strings.HasPrefix(model, "vm-"):
			return EntityFirewall, nil
		case strings.HasPrefix(model, "m-") || strings.Contains(model, "panorama"):
			return EntityPanorama, nil
		}
	}
	if strings.Contains(joined, "panorama") {
		return EntityPanorama, nil
	}
	if strings.Contains(joined, "firewall") || strings.Contains(joined, "pan-os") {
		return EntityFirewall, nil
	}
	return "", ErrParseFatal
}

func firstSerial(files map[string]string) string {
	paths := sortedKeys(files)
	for _, path := range paths {
		m := reSerial.FindStringSubmatch(files[path])
		if len(m) == 2 {
			return m[1]
		}
	}
	return ""
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func joinContent(files map[string]string) string {
	parts := make([]string, 0, len(files))
	for _, k := range sortedKeys(files) {
		parts = append(parts, files[k])
	}
	return strings.Join(parts, "\n")
}
