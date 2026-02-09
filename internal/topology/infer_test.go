package topology

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInferSharedSubnetEdges(t *testing.T) {
	ifaces := []Interface{
		{DeviceID: "fw-a", Zone: "trust", VR: "vr1", Name: "eth1", IPCIDRs: []string{"10.0.0.1/24"}},
		{DeviceID: "fw-b", Zone: "inside", VR: "vr1", Name: "eth2", IPCIDRs: []string{"10.0.0.2/24"}},
		{DeviceID: "fw-c", Zone: "dmz", VR: "vr2", Name: "eth3", IPCIDRs: []string{"10.0.0.3/24"}},
		{DeviceID: "fw-d", Zone: "v6", VR: "vr1", Name: "eth4", IPCIDRs: []string{"2001:db8::1/64"}},
	}

	edges := InferSharedSubnetEdges("prod", ifaces)
	if len(edges) != 1 {
		t.Fatalf("len(edges)=%d, want 1", len(edges))
	}
	e := edges[0]
	if e.EdgeType != "shared_subnet" || e.Source != "inferred" {
		t.Fatalf("unexpected edge type/source: %#v", e)
	}
	if e.SrcVR != "vr1" || e.DstVR != "vr1" {
		t.Fatalf("edge must be VR-aware and matching: %#v", e)
	}
	if e.SrcDeviceID == "fw-c" || e.DstDeviceID == "fw-c" {
		t.Fatalf("different VR should not connect: %#v", e)
	}
	if e.SrcDeviceID == "fw-d" || e.DstDeviceID == "fw-d" {
		t.Fatalf("ipv6-only interface should not create inferred edge: %#v", e)
	}
}

func TestOverrideMerge(t *testing.T) {
	root := t.TempDir()
	overridePath := filepath.Join(root, "topology_links.json")
	data := `[
  {"src_device_id":"fw-c","src_zone":"dmz","src_interface":"eth7","src_vr":"vr3","dst_device_id":"fw-d","dst_zone":"edge","dst_interface":"eth8","dst_vr":"vr3"},
  {"src_device_id":"fw-a","src_zone":"trust","src_interface":"eth1","src_vr":"vr1","dst_device_id":"fw-b","dst_zone":"inside","dst_interface":"eth2","dst_vr":"vr1"}
]`
	if err := os.WriteFile(overridePath, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	inferred := []Edge{
		makeEdge("shared_subnet", "inferred",
			Interface{DeviceID: "fw-a", Zone: "trust", Name: "eth1", VR: "vr1"},
			Interface{DeviceID: "fw-b", Zone: "inside", Name: "eth2", VR: "vr1"},
		),
	}

	merged, err := MergeOverrideEdges("prod", inferred, overridePath)
	if err != nil {
		t.Fatalf("MergeOverrideEdges() err=%v", err)
	}
	if len(merged) != 3 {
		t.Fatalf("len(merged)=%d, want 3 (1 inferred + 2 overrides)", len(merged))
	}

	foundOverride := false
	for i := 1; i < len(merged); i++ {
		if merged[i-1].EdgeID > merged[i].EdgeID {
			t.Fatalf("edges not sorted by edge_id")
		}
	}
	for _, e := range merged {
		if e.EdgeType == "manual_override" {
			foundOverride = true
			if e.Source != "override" {
				t.Fatalf("manual override source mismatch: %#v", e)
			}
		}
	}
	if !foundOverride {
		t.Fatal("expected at least one manual_override edge")
	}
}
