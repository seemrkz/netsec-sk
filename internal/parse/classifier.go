package parse

import (
	"errors"
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
		return ParseOutput{Result: "parse_error_fatal"}, err
	}

	switch entityType {
	case EntityFirewall:
		out, partial, err := ParseFirewallSnapshot(ctx, files)
		if err != nil {
			return ParseOutput{Result: "parse_error_fatal"}, err
		}
		result := "ok"
		if partial {
			result = "parse_error_partial"
		}
		return ParseOutput{EntityType: entityType, EntityID: out["device"].(map[string]any)["id"].(string), Snapshot: out, Result: result}, nil
	case EntityPanorama:
		out, partial, err := ParsePanoramaSnapshot(ctx, files)
		if err != nil {
			return ParseOutput{Result: "parse_error_fatal"}, err
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
