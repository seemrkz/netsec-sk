package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
)

type flowTraceRequest struct {
	SrcIP string `json:"src_ip"`
	DstIP string `json:"dst_ip"`
}

type flowHop struct {
	Index           int    `json:"index"`
	LogicalDeviceID string `json:"logical_device_id"`
	Hostname        string `json:"hostname"`
	IngressZone     string `json:"ingress_zone"`
	EgressZone      string `json:"egress_zone"`
	UsedDefault     bool   `json:"used_default"`
}

func (a *app) handleFlowTrace(w http.ResponseWriter, r *http.Request, envID string) {
	envDir, status := a.resolveEnvironmentPath(envID)
	if status != http.StatusOK {
		if status == http.StatusGone {
			writeError(w, http.StatusNotFound, "ERR_ENV_ALREADY_DELETED", "environment already deleted")
			return
		}
		writeError(w, http.StatusNotFound, "ERR_ENV_NOT_FOUND", "environment not found")
		return
	}

	defer r.Body.Close()
	var req flowTraceRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_IP", "invalid request body")
		return
	}
	src, err1 := netip.ParseAddr(strings.TrimSpace(req.SrcIP))
	dst, err2 := netip.ParseAddr(strings.TrimSpace(req.DstIP))
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_IP", "src_ip and dst_ip must be valid IP literals")
		return
	}

	state, err := loadState(envDir)
	if err != nil {
		writeError(w, http.StatusNotFound, "ERR_ENV_STATE_NOT_FOUND", "environment state not found")
		return
	}

	devs := logicalDevices(state)
	srcDev := resolveSourceFirewall(devs, src)
	if srcDev == nil {
		writeError(w, http.StatusNotFound, "ERR_FLOW_SRC_NOT_FOUND", "source firewall not found")
		return
	}
	dstDev := resolveDestinationFirewall(devs, dst)
	if dstDev == nil {
		writeError(w, http.StatusNotFound, "ERR_FLOW_PATH_NOT_FOUND", "destination firewall not found")
		return
	}

	srcID := valueString(srcDev["logical_device_id"], "")
	dstID := valueString(dstDev["logical_device_id"], "")
	path := findPathByTopology(state, srcID, dstID)
	if len(path) == 0 {
		writeError(w, http.StatusNotFound, "ERR_FLOW_PATH_NOT_FOUND", "no deterministic path found")
		return
	}

	hops := make([]flowHop, 0, len(path))
	for i, id := range path {
		dev := findDeviceByID(devs, id)
		cur, _ := dev["current"].(map[string]any)
		identity, _ := cur["identity"].(map[string]any)
		egressZone, usedDefault := resolveEgressZone(dev, dst)
		hops = append(hops, flowHop{
			Index:           i,
			LogicalDeviceID: id,
			Hostname:        valueString(identity["hostname"], "not_found"),
			IngressZone:     "not_found",
			EgressZone:      egressZone,
			UsedDefault:     usedDefault,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"env_id":  envID,
		"src_ip":  src.String(),
		"dst_ip":  dst.String(),
		"hops":    hops,
		"mermaid": buildMermaid(hops),
	})
}

func resolveSourceFirewall(devs []map[string]any, src netip.Addr) map[string]any {
	connected := make([]map[string]any, 0)
	for _, d := range devs {
		if valueString(d["device_type"], "") != "firewall" {
			continue
		}
		if deviceContainsIPInInterfaces(d, src) {
			connected = append(connected, d)
		}
	}
	if len(connected) > 0 {
		sort.Slice(connected, func(i, j int) bool {
			return valueString(connected[i]["logical_device_id"], "") < valueString(connected[j]["logical_device_id"], "")
		})
		return connected[0]
	}

	bestBits := -1
	var best map[string]any
	for _, d := range devs {
		if valueString(d["device_type"], "") != "firewall" {
			continue
		}
		r := longestRouteMatch(d, src, false)
		if r.bits < 0 {
			continue
		}
		if r.bits > bestBits || (r.bits == bestBits && best != nil &&
			valueString(d["logical_device_id"], "") < valueString(best["logical_device_id"], "")) {
			bestBits = r.bits
			best = d
		}
	}
	return best
}

func resolveDestinationFirewall(devs []map[string]any, dst netip.Addr) map[string]any {
	bestBits := -1
	var best map[string]any
	for _, d := range devs {
		if valueString(d["device_type"], "") != "firewall" {
			continue
		}
		r := longestRouteMatch(d, dst, true)
		if r.bits < 0 {
			continue
		}
		if r.bits > bestBits || (r.bits == bestBits && best != nil &&
			valueString(d["logical_device_id"], "") < valueString(best["logical_device_id"], "")) {
			bestBits = r.bits
			best = d
		}
	}
	return best
}

type routeMatch struct {
	bits        int
	zone        string
	usedDefault bool
}

func resolveEgressZone(dev map[string]any, dst netip.Addr) (string, bool) {
	r := longestRouteMatch(dev, dst, false)
	if r.bits >= 0 {
		return r.zone, r.usedDefault
	}
	r = longestRouteMatch(dev, dst, true)
	if r.bits >= 0 {
		return r.zone, r.usedDefault
	}
	return "not_found", false
}

func longestRouteMatch(dev map[string]any, ip netip.Addr, includeDefault bool) routeMatch {
	cur, _ := dev["current"].(map[string]any)
	network, _ := cur["network"].(map[string]any)
	cands := append(toAnySlice(network["routes_runtime"]), toAnySlice(network["routes_config"])...)
	best := routeMatch{bits: -1, zone: "not_found", usedDefault: false}
	for _, it := range cands {
		r, _ := it.(map[string]any)
		dst := valueString(r["destination"], "")
		pfx, err := netip.ParsePrefix(dst)
		if err != nil {
			continue
		}
		if dst == "0.0.0.0/0" && !includeDefault {
			continue
		}
		if !pfx.Contains(ip) {
			continue
		}
		if pfx.Bits() > best.bits {
			best = routeMatch{bits: pfx.Bits(), zone: valueString(r["zone"], "not_found"), usedDefault: dst == "0.0.0.0/0"}
		}
	}
	return best
}

func deviceContainsIPInInterfaces(dev map[string]any, ip netip.Addr) bool {
	cur, _ := dev["current"].(map[string]any)
	network, _ := cur["network"].(map[string]any)
	ifaces := toAnySlice(network["interfaces"])
	for _, it := range ifaces {
		iface, _ := it.(map[string]any)
		units := toAnySlice(iface["layer3_units"])
		for _, u := range units {
			unit, _ := u.(map[string]any)
			for _, cidrAny := range toAnySlice(unit["ip_cidrs"]) {
				cidr := valueString(cidrAny, "")
				pfx, err := netip.ParsePrefix(cidr)
				if err == nil && pfx.Contains(ip) {
					return true
				}
			}
		}
	}
	return false
}

func findPathByTopology(state map[string]any, srcID, dstID string) []string {
	if srcID == dstID {
		return []string{srcID}
	}
	topology, _ := state["topology"].(map[string]any)
	edges := toAnySlice(topology["inferred_adjacencies"])
	adj := map[string][]string{}
	for _, e := range edges {
		m, _ := e.(map[string]any)
		aID := valueString(m["fw_a_logical_device_id"], "")
		bID := valueString(m["fw_b_logical_device_id"], "")
		if aID == "" || bID == "" {
			continue
		}
		adj[aID] = append(adj[aID], bID)
		adj[bID] = append(adj[bID], aID)
	}
	for k := range adj {
		sort.Strings(adj[k])
	}

	type node struct {
		id   string
		path []string
	}
	q := []node{{id: srcID, path: []string{srcID}}}
	seen := map[string]bool{srcID: true}
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		for _, nxt := range adj[n.id] {
			if seen[nxt] {
				continue
			}
			path := append(append([]string{}, n.path...), nxt)
			if nxt == dstID {
				return path
			}
			seen[nxt] = true
			q = append(q, node{id: nxt, path: path})
		}
	}
	return nil
}

func findDeviceByID(devs []map[string]any, id string) map[string]any {
	for _, d := range devs {
		if valueString(d["logical_device_id"], "") == id {
			return d
		}
	}
	return map[string]any{}
}

func buildMermaid(hops []flowHop) string {
	lines := []string{"flowchart LR"}
	for i, h := range hops {
		label := h.Hostname
		if label == "" || label == "not_found" {
			label = h.LogicalDeviceID
		}
		lines = append(lines, "  N"+itoa(i)+"[\""+escapeMermaid(label)+"\"]")
		if i > 0 {
			lines = append(lines, "  N"+itoa(i-1)+" --> N"+itoa(i))
		}
	}
	return strings.Join(lines, "\n")
}

func escapeMermaid(s string) string {
	s = strings.ReplaceAll(s, "\"", "'")
	return s
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
