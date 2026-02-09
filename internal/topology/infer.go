package topology

import (
	"encoding/json"
	"net"
	"os"
	"sort"
)

type Interface struct {
	DeviceID string
	Zone     string
	VR       string
	Name     string
	IPCIDRs  []string
}

type Edge struct {
	EdgeID       string
	EdgeType     string
	SrcDeviceID  string
	SrcZone      string
	SrcInterface string
	SrcVR        string
	DstDeviceID  string
	DstZone      string
	DstInterface string
	DstVR        string
	Source       string
}

func InferSharedSubnetEdges(envID string, ifaces []Interface) []Edge {
	edges := make([]Edge, 0)
	seen := map[string]struct{}{}

	for i := 0; i < len(ifaces); i++ {
		for j := i + 1; j < len(ifaces); j++ {
			a := ifaces[i]
			b := ifaces[j]
			if a.DeviceID == b.DeviceID {
				continue
			}
			if a.VR == "" || b.VR == "" || a.VR != b.VR {
				continue
			}

			if !sharesIPv4Subnet(a.IPCIDRs, b.IPCIDRs) {
				continue
			}

			e := makeEdge("shared_subnet", "inferred", a, b)
			if _, ok := seen[e.EdgeID]; ok {
				continue
			}
			seen[e.EdgeID] = struct{}{}
			edges = append(edges, e)
		}
	}

	sort.Slice(edges, func(i, j int) bool { return edges[i].EdgeID < edges[j].EdgeID })
	return edges
}

type OverrideEdge struct {
	SrcDeviceID  string `json:"src_device_id"`
	SrcZone      string `json:"src_zone"`
	SrcInterface string `json:"src_interface"`
	SrcVR        string `json:"src_vr"`
	DstDeviceID  string `json:"dst_device_id"`
	DstZone      string `json:"dst_zone"`
	DstInterface string `json:"dst_interface"`
	DstVR        string `json:"dst_vr"`
}

func MergeOverrideEdges(envID string, inferred []Edge, overridePath string) ([]Edge, error) {
	_ = envID
	edges := make([]Edge, 0, len(inferred))
	edges = append(edges, inferred...)

	data, err := os.ReadFile(overridePath)
	if err != nil {
		if os.IsNotExist(err) {
			sort.Slice(edges, func(i, j int) bool { return edges[i].EdgeID < edges[j].EdgeID })
			return edges, nil
		}
		return nil, err
	}

	var overrides []OverrideEdge
	if err := json.Unmarshal(data, &overrides); err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	for _, e := range edges {
		seen[e.EdgeID] = struct{}{}
	}
	for _, o := range overrides {
		a := Interface{DeviceID: o.SrcDeviceID, Zone: o.SrcZone, VR: o.SrcVR, Name: o.SrcInterface}
		b := Interface{DeviceID: o.DstDeviceID, Zone: o.DstZone, VR: o.DstVR, Name: o.DstInterface}
		e := makeEdge("manual_override", "override", a, b)
		if _, ok := seen[e.EdgeID]; ok {
			continue
		}
		seen[e.EdgeID] = struct{}{}
		edges = append(edges, e)
	}

	sort.Slice(edges, func(i, j int) bool { return edges[i].EdgeID < edges[j].EdgeID })
	return edges, nil
}

func makeEdge(edgeType string, source string, a Interface, b Interface) Edge {
	left, right := a, b
	if edgeEndpointKey(a) > edgeEndpointKey(b) {
		left, right = b, a
	}
	edgeID := edgeType + "|" + edgeEndpointKey(left) + "|" + edgeEndpointKey(right)

	return Edge{
		EdgeID:       edgeID,
		EdgeType:     edgeType,
		SrcDeviceID:  left.DeviceID,
		SrcZone:      left.Zone,
		SrcInterface: left.Name,
		SrcVR:        left.VR,
		DstDeviceID:  right.DeviceID,
		DstZone:      right.Zone,
		DstInterface: right.Name,
		DstVR:        right.VR,
		Source:       source,
	}
}

func edgeEndpointKey(i Interface) string {
	return i.DeviceID + "|" + i.Zone + "|" + i.Name + "|" + i.VR
}

func sharesIPv4Subnet(aCIDRs []string, bCIDRs []string) bool {
	for _, a := range aCIDRs {
		aIP, aNet, ok := parseIPv4CIDR(a)
		if !ok {
			continue
		}
		for _, b := range bCIDRs {
			bIP, bNet, ok := parseIPv4CIDR(b)
			if !ok {
				continue
			}
			if sameMask(aNet, bNet) && aNet.Contains(bIP) && bNet.Contains(aIP) {
				return true
			}
		}
	}
	return false
}

func parseIPv4CIDR(cidr string) (net.IP, *net.IPNet, bool) {
	ip, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, nil, false
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return nil, nil, false
	}
	return ip4, n, true
}

func sameMask(a *net.IPNet, b *net.IPNet) bool {
	ao, ab := a.Mask.Size()
	bo, bb := b.Mask.Size()
	return ao == bo && ab == bb
}
