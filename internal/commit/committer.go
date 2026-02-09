package commit

import (
	"path/filepath"
	"sort"
	"strings"
)

type Meta struct {
	EnvID      string
	EntityType string
	EntityID   string
	StateSHA   string
	TSFID      string
}

func BuildCommitSubject(m Meta) string {
	short := m.StateSHA
	if len(short) > 12 {
		short = short[:12]
	}
	tsf := strings.ReplaceAll(m.TSFID, " ", "_")
	return "ingest(" + m.EnvID + "): " + m.EntityType + "/" + m.EntityID + " " + short + " " + tsf
}

func BuildAllowlist(repoPath, envID, entityType, entityID, snapshotFile string) []string {
	base := filepath.Join(repoPath, "envs", envID)
	entityDir := "devices"
	if entityType == "panorama" {
		entityDir = "panorama"
	}

	out := []string{
		filepath.Join(base, "state", "commits.ndjson"),
		filepath.Join(base, "state", entityDir, entityID, "latest.json"),
		filepath.Join(base, "state", entityDir, entityID, "snapshots", snapshotFile),
		filepath.Join(base, "exports", "environment.json"),
		filepath.Join(base, "exports", "inventory.csv"),
		filepath.Join(base, "exports", "nodes.csv"),
		filepath.Join(base, "exports", "edges.csv"),
		filepath.Join(base, "exports", "topology.mmd"),
		filepath.Join(base, "exports", "agent_context.md"),
	}
	sort.Strings(out)
	return out
}
