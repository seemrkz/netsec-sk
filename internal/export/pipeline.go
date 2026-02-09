package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/seemrkz/netsec-sk/internal/topology"
)

type RunOptions struct {
	RepoPath string
	EnvID    string
	Now      time.Time
}

func Run(opts RunOptions) error {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	envBase := filepath.Join(opts.RepoPath, "envs", opts.EnvID)
	stateBase := filepath.Join(envBase, "state")
	exportBase := filepath.Join(envBase, "exports")

	firewalls, fwInv, fwIfaces, zoneSet, fwNodes, err := loadFirewalls(stateBase, opts.EnvID)
	if err != nil {
		return err
	}
	panoramas, panoInv, panoNodes, err := loadPanorama(stateBase, opts.EnvID)
	if err != nil {
		return err
	}
	invRows := append(fwInv, panoInv...)

	edges := topology.InferSharedSubnetEdges(opts.EnvID, fwIfaces)
	overridePath := filepath.Join(envBase, "overrides", "topology_links.json")
	edges, err = topology.MergeOverrideEdges(opts.EnvID, edges, overridePath)
	if err != nil {
		return err
	}

	edgeCSVRows := make([][]string, 0, len(edges))
	zoneNodes := make(map[string]nodeRow)
	mermaidEdges := make([]EdgeRow, 0, len(edges))
	for _, e := range edges {
		srcNode := zoneNodeID(e.SrcDeviceID, e.SrcZone)
		dstNode := zoneNodeID(e.DstDeviceID, e.DstZone)
		zoneNodes[srcNode] = nodeRow{NodeID: srcNode, NodeType: "zone", EnvID: opts.EnvID, DeviceID: e.SrcDeviceID, Zone: e.SrcZone, VR: e.SrcVR, Label: e.SrcDeviceID + ":" + e.SrcZone}
		zoneNodes[dstNode] = nodeRow{NodeID: dstNode, NodeType: "zone", EnvID: opts.EnvID, DeviceID: e.DstDeviceID, Zone: e.DstZone, VR: e.DstVR, Label: e.DstDeviceID + ":" + e.DstZone}
		edgeCSVRows = append(edgeCSVRows, []string{
			e.EdgeID, e.EdgeType, srcNode, dstNode, e.SrcDeviceID, e.SrcZone, e.SrcInterface, e.SrcVR, e.DstDeviceID, e.DstZone, e.DstInterface, e.DstVR, "", e.Source,
		})
		mermaidEdges = append(mermaidEdges, EdgeRow{EdgeID: e.EdgeID, Src: srcNode, Dst: dstNode})
	}

	nodes := append(fwNodes, panoNodes...)
	for _, n := range zoneNodes {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].NodeType == nodes[j].NodeType {
			return nodes[i].NodeID < nodes[j].NodeID
		}
		return nodes[i].NodeType < nodes[j].NodeType
	})
	sort.Slice(edgeCSVRows, func(i, j int) bool { return edgeCSVRows[i][0] < edgeCSVRows[j][0] })

	envJSON, err := BuildEnvironmentJSON(
		opts.EnvID,
		now.Format(time.RFC3339),
		Counts{Firewalls: len(firewalls), Panorama: len(panoramas), Zones: len(zoneSet), TopologyEdges: len(edges)},
		firewalls,
		panoramas,
		toZoneEdgeObjects(edges),
	)
	if err != nil {
		return err
	}
	inventoryCSV, err := BuildInventoryCSV(invRows)
	if err != nil {
		return err
	}
	nodesCSV, err := BuildCSV(
		[]string{"node_id", "node_type", "env_id", "device_id", "panorama_id", "zone", "virtual_router", "label"},
		toNodeRows(nodes),
	)
	if err != nil {
		return err
	}
	edgesCSV, err := BuildCSV(
		[]string{"edge_id", "edge_type", "src_node_id", "dst_node_id", "src_device_id", "src_zone", "src_interface", "src_vr", "dst_device_id", "dst_zone", "dst_interface", "dst_vr", "evidence", "source"},
		edgeCSVRows,
	)
	if err != nil {
		return err
	}
	mermaid := BuildTopologyMermaid(mermaidEdges)
	agentCtx := BuildAgentContextMarkdown(AgentContext{
		EnvironmentSummary: "Environment: " + opts.EnvID,
		InventoryCounts:    "Firewalls: " + itoa(len(firewalls)) + ", Panorama: " + itoa(len(panoramas)),
		RoutingUsage:       "See inventory.csv for routing protocol usage.",
		PanoramaOverview:   "Panorama instances: " + itoa(len(panoramas)),
		TopologyHighlights: "Topology edges: " + itoa(len(edges)),
		OrphansAndUnknowns: "Zone count: " + itoa(len(zoneSet)),
	})

	if err := os.MkdirAll(exportBase, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(exportBase, "environment.json"), append(envJSON, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(exportBase, "inventory.csv"), []byte(inventoryCSV), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(exportBase, "nodes.csv"), []byte(nodesCSV), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(exportBase, "edges.csv"), []byte(edgesCSV), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(exportBase, "topology.mmd"), []byte(mermaid), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(exportBase, "agent_context.md"), []byte(agentCtx), 0o644); err != nil {
		return err
	}
	return nil
}

type nodeRow struct {
	NodeID     string
	NodeType   string
	EnvID      string
	DeviceID   string
	PanoramaID string
	Zone       string
	VR         string
	Label      string
}

func toNodeRows(rows []nodeRow) [][]string {
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{r.NodeID, r.NodeType, r.EnvID, r.DeviceID, r.PanoramaID, r.Zone, r.VR, r.Label})
	}
	return out
}

func toZoneEdgeObjects(edges []topology.Edge) []map[string]any {
	out := make([]map[string]any, 0, len(edges))
	for _, e := range edges {
		out = append(out, map[string]any{
			"edge_id": e.EdgeID,
			"src": map[string]any{
				"device_id": e.SrcDeviceID,
				"zone":      e.SrcZone,
				"interface": e.SrcInterface,
				"vr":        e.SrcVR,
			},
			"dst": map[string]any{
				"device_id": e.DstDeviceID,
				"zone":      e.DstZone,
				"interface": e.DstInterface,
				"vr":        e.DstVR,
			},
			"edge_type": e.EdgeType,
			"source":    e.Source,
		})
	}
	return out
}

func loadFirewalls(stateBase, envID string) ([]map[string]any, []InventoryRow, []topology.Interface, map[string]struct{}, []nodeRow, error) {
	root := filepath.Join(stateBase, "devices")
	entries, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, nil, nil, nil, err
	}
	firewalls := make([]map[string]any, 0)
	invRows := make([]InventoryRow, 0)
	ifaces := make([]topology.Interface, 0)
	zoneSet := make(map[string]struct{})
	nodes := make([]nodeRow, 0)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		latestPath := filepath.Join(root, e.Name(), "latest.json")
		doc, ok, err := readJSONMap(latestPath)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if !ok {
			continue
		}
		firewalls = append(firewalls, doc)
		device, _ := doc["device"].(map[string]any)
		source, _ := doc["source"].(map[string]any)
		ha, _ := doc["ha"].(map[string]any)
		nodes = append(nodes, nodeRow{
			NodeID:   firewallNodeID(e.Name()),
			NodeType: "firewall",
			EnvID:    envID,
			DeviceID: e.Name(),
			Label:    stringField(device["hostname"]),
		})
		invRows = append(invRows, InventoryRow{
			EntityType:        "firewall",
			EntityID:          e.Name(),
			Hostname:          stringField(device["hostname"]),
			Serial:            stringField(device["serial"]),
			Model:             stringField(device["model"]),
			Version:           stringField(device["sw_version"]),
			MgmtIP:            stringField(device["mgmt_ip"]),
			HAEnabled:         boolString(ha["enabled"]),
			HAMode:            stringField(ha["mode"]),
			HAState:           stringField(ha["local_state"]),
			SourceTSFID:       stringField(source["tsf_id"]),
			StateSHA256:       stringField(doc["state_sha256"]),
			LastIngestedAtUTC: stringField(source["ingested_at_utc"]),
		})
		network, _ := doc["network"].(map[string]any)
		intfs, _ := network["interfaces"].([]any)
		for _, raw := range intfs {
			m, _ := raw.(map[string]any)
			zone := stringField(m["zone"])
			vr := stringField(m["virtual_router"])
			zoneSet[e.Name()+"|"+zone] = struct{}{}
			ifaces = append(ifaces, topology.Interface{
				DeviceID: e.Name(),
				Zone:     zone,
				VR:       vr,
				Name:     stringField(m["name"]),
				IPCIDRs:  toStringSlice(m["ip_cidrs"]),
			})
		}
	}
	sort.Slice(firewalls, func(i, j int) bool {
		return stringField(firewalls[i]["state_sha256"]) < stringField(firewalls[j]["state_sha256"])
	})
	return firewalls, invRows, ifaces, zoneSet, nodes, nil
}

func loadPanorama(stateBase, envID string) ([]map[string]any, []InventoryRow, []nodeRow, error) {
	root := filepath.Join(stateBase, "panorama")
	entries, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, nil, err
	}
	panoramas := make([]map[string]any, 0)
	invRows := make([]InventoryRow, 0)
	nodes := make([]nodeRow, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		latestPath := filepath.Join(root, e.Name(), "latest.json")
		doc, ok, err := readJSONMap(latestPath)
		if err != nil {
			return nil, nil, nil, err
		}
		if !ok {
			continue
		}
		panoramas = append(panoramas, doc)
		inst, _ := doc["panorama_instance"].(map[string]any)
		source, _ := doc["source"].(map[string]any)
		nodes = append(nodes, nodeRow{
			NodeID:     panoramaNodeID(e.Name()),
			NodeType:   "panorama",
			EnvID:      envID,
			PanoramaID: e.Name(),
			Label:      stringField(inst["hostname"]),
		})
		invRows = append(invRows, InventoryRow{
			EntityType:        "panorama",
			EntityID:          e.Name(),
			Hostname:          stringField(inst["hostname"]),
			Serial:            stringField(inst["serial"]),
			Model:             stringField(inst["model"]),
			Version:           stringField(inst["version"]),
			MgmtIP:            stringField(inst["mgmt_ip"]),
			SourceTSFID:       stringField(source["tsf_id"]),
			StateSHA256:       stringField(doc["state_sha256"]),
			LastIngestedAtUTC: stringField(source["ingested_at_utc"]),
		})
	}
	sort.Slice(panoramas, func(i, j int) bool {
		return stringField(panoramas[i]["state_sha256"]) < stringField(panoramas[j]["state_sha256"])
	})
	return panoramas, invRows, nodes, nil
}

func readJSONMap(path string) (map[string]any, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func firewallNodeID(id string) string { return "firewall_" + sanitize(id) }
func panoramaNodeID(id string) string { return "panorama_" + sanitize(id) }
func zoneNodeID(deviceID, zone string) string {
	return "zone_" + sanitize(deviceID) + "_" + sanitize(zone)
}
func sanitize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	repl := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_", ".", "_", "-", "_")
	return repl.Replace(s)
}

func toStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, it := range arr {
		out = append(out, stringField(it))
	}
	sort.Strings(out)
	return out
}
func stringField(v any) string {
	s, _ := v.(string)
	return s
}
func boolString(v any) string {
	b, _ := v.(bool)
	if b {
		return "true"
	}
	return "false"
}
func itoa(v int) string {
	return strconv.Itoa(v)
}
