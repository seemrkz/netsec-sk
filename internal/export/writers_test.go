package export

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEnvironmentJSONSchema(t *testing.T) {
	out, err := BuildEnvironmentJSON(
		"prod",
		"2026-02-09T00:00:00Z",
		Counts{Firewalls: 2, Panorama: 1, Zones: 3, TopologyEdges: 4},
		[]map[string]any{},
		[]map[string]any{},
		[]map[string]any{},
	)
	if err != nil {
		t.Fatalf("BuildEnvironmentJSON() err=%v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("json unmarshal err=%v", err)
	}
	if doc["schema_version"] != float64(1) {
		t.Fatalf("schema_version=%v", doc["schema_version"])
	}
	env, ok := doc["environment"].(map[string]any)
	if !ok {
		t.Fatal("missing environment object")
	}
	if env["env_id"] != "prod" {
		t.Fatalf("unexpected environment fields: %#v", env)
	}
	if _, ok := env["counts"].(map[string]any); !ok {
		t.Fatal("missing counts object")
	}
	if _, ok := doc["firewalls"]; !ok {
		t.Fatal("missing firewalls key")
	}
	if _, ok := doc["panorama"]; !ok {
		t.Fatal("missing panorama key")
	}
	topology, ok := doc["topology"].(map[string]any)
	if !ok {
		t.Fatal("missing topology object")
	}
	if _, ok := topology["zone_edges"]; !ok {
		t.Fatal("missing topology.zone_edges")
	}
}

func TestCSVHeadersAndOrdering(t *testing.T) {
	rows := []InventoryRow{
		{
			EntityType:                 "panorama",
			EntityID:                   "p2",
			Hostname:                   "p2",
			RoutingProtocolsConfigured: []string{"bgp", "static"},
			RoutingProtocolsActive:     []string{"ospf", "bgp"},
		},
		{
			EntityType:                 "firewall",
			EntityID:                   "f1",
			Hostname:                   "f1",
			RoutingProtocolsConfigured: []string{"ospf", "bgp"},
			RoutingProtocolsActive:     []string{"static", "bgp"},
		},
	}

	csvText, err := BuildInventoryCSV(rows)
	if err != nil {
		t.Fatalf("BuildInventoryCSV() err=%v", err)
	}

	lines := strings.Split(strings.TrimSpace(csvText), "\n")
	if len(lines) != 3 {
		t.Fatalf("line count=%d, want 3", len(lines))
	}
	wantHeader := "entity_type,entity_id,hostname,serial,model,version,mgmt_ip,ha_enabled,ha_mode,ha_state,routing_protocols_configured,routing_protocols_active,source_tsf_id,state_sha256,last_ingested_at_utc"
	if lines[0] != wantHeader {
		t.Fatalf("header mismatch:\n got=%s\nwant=%s", lines[0], wantHeader)
	}
	if !strings.HasPrefix(lines[1], "firewall,f1,") {
		t.Fatalf("row ordering mismatch, expected firewall first: %s", lines[1])
	}
	if !strings.Contains(lines[1], "bgp;ospf") || !strings.Contains(lines[1], "bgp;static") {
		t.Fatalf("protocol sorting/format mismatch in row: %s", lines[1])
	}
}

func TestAgentContextTemplate(t *testing.T) {
	md := BuildAgentContextMarkdown(AgentContext{
		EnvironmentSummary: "Env summary",
		InventoryCounts:    "Counts",
		RoutingUsage:       "Routing",
		PanoramaOverview:   "Panorama",
		TopologyHighlights: "Highlights",
		OrphansAndUnknowns: "Unknowns",
	})

	wantOrder := []string{"# Environment Summary", "## Inventory Counts", "## Routing Usage", "## Panorama Overview", "## Topology Highlights", "## Orphans and Unknowns"}
	lastIndex := -1
	for _, heading := range wantOrder {
		idx := strings.Index(md, heading)
		if idx < 0 {
			t.Fatalf("missing heading: %s", heading)
		}
		if idx <= lastIndex {
			t.Fatalf("heading order mismatch at: %s", heading)
		}
		lastIndex = idx
	}
}
