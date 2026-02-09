package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Counts struct {
	Firewalls     int
	Panorama      int
	Zones         int
	TopologyEdges int
}

type InventoryRow struct {
	EntityType                 string
	EntityID                   string
	Hostname                   string
	Serial                     string
	Model                      string
	Version                    string
	MgmtIP                     string
	HAEnabled                  string
	HAMode                     string
	HAState                    string
	RoutingProtocolsConfigured []string
	RoutingProtocolsActive     []string
	SourceTSFID                string
	StateSHA256                string
	LastIngestedAtUTC          string
}

type EdgeRow struct {
	EdgeID string
	Src    string
	Dst    string
}

type AgentContext struct {
	EnvironmentSummary string
	InventoryCounts    string
	RoutingUsage       string
	PanoramaOverview   string
	TopologyHighlights string
	OrphansAndUnknowns string
}

func BuildEnvironmentJSON(envID string, generatedAtUTC string, counts Counts, firewalls []map[string]any, panorama []map[string]any, zoneEdges []map[string]any) ([]byte, error) {
	doc := map[string]any{
		"schema_version": 1,
		"environment": map[string]any{
			"env_id":           envID,
			"generated_at_utc": generatedAtUTC,
			"counts": map[string]any{
				"firewalls":      counts.Firewalls,
				"panorama":       counts.Panorama,
				"zones":          counts.Zones,
				"topology_edges": counts.TopologyEdges,
			},
		},
		"firewalls": firewalls,
		"panorama":  panorama,
		"topology": map[string]any{
			"zone_edges": zoneEdges,
		},
	}
	return json.MarshalIndent(doc, "", "  ")
}

func BuildInventoryCSV(rows []InventoryRow) (string, error) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].EntityType == rows[j].EntityType {
			return rows[i].EntityID < rows[j].EntityID
		}
		return rows[i].EntityType < rows[j].EntityType
	})

	header := []string{
		"entity_type", "entity_id", "hostname", "serial", "model", "version", "mgmt_ip", "ha_enabled", "ha_mode", "ha_state", "routing_protocols_configured", "routing_protocols_active", "source_tsf_id", "state_sha256", "last_ingested_at_utc",
	}

	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	if err := w.Write(header); err != nil {
		return "", err
	}
	for _, r := range rows {
		cfg := append([]string{}, r.RoutingProtocolsConfigured...)
		act := append([]string{}, r.RoutingProtocolsActive...)
		sort.Strings(cfg)
		sort.Strings(act)
		if err := w.Write([]string{
			r.EntityType, r.EntityID, r.Hostname, r.Serial, r.Model, r.Version, r.MgmtIP, r.HAEnabled, r.HAMode, r.HAState, strings.Join(cfg, ";"), strings.Join(act, ";"), r.SourceTSFID, r.StateSHA256, r.LastIngestedAtUTC,
		}); err != nil {
			return "", err
		}
	}
	w.Flush()
	return buf.String(), w.Error()
}

func BuildTopologyMermaid(edges []EdgeRow) string {
	sort.Slice(edges, func(i, j int) bool { return edges[i].EdgeID < edges[j].EdgeID })
	lines := []string{"graph TD"}
	for _, e := range edges {
		lines = append(lines, fmt.Sprintf("  %s --> %s", sanitizeNodeID(e.Src), sanitizeNodeID(e.Dst)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildAgentContextMarkdown(ctx AgentContext) string {
	return strings.Join([]string{
		"# Environment Summary",
		ctx.EnvironmentSummary,
		"",
		"## Inventory Counts",
		ctx.InventoryCounts,
		"",
		"## Routing Usage",
		ctx.RoutingUsage,
		"",
		"## Panorama Overview",
		ctx.PanoramaOverview,
		"",
		"## Topology Highlights",
		ctx.TopologyHighlights,
		"",
		"## Orphans and Unknowns",
		ctx.OrphansAndUnknowns,
		"",
	}, "\n")
}

func BuildCSV(header []string, rows [][]string) (string, error) {
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	if err := w.Write(header); err != nil {
		return "", err
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return "", err
		}
	}
	w.Flush()
	return buf.String(), w.Error()
}

func sanitizeNodeID(in string) string {
	var b strings.Builder
	for _, r := range in {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
