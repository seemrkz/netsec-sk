package tsf

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	rePattern1 = regexp.MustCompile(`(^|.*/)tmp/cli/[^/]+\.txt$`)
	rePattern2 = regexp.MustCompile(`(^|.*/)tmp/cli/.*\.txt$`)
	reTSStart  = regexp.MustCompile(`^[A-Za-z0-9._-]+_ts\.(?:tgz|tar\.gz)`)
	reSerial   = regexp.MustCompile(`(?im)^\s*(?:serial|serial number|device serial)\s*:\s*(\S+)`)
)

type Identity struct {
	TSFID           string
	TSFOriginalName string
	Serial          string
}

type ReadFileFunc func(path string) ([]byte, error)

func DeriveIdentity(paths []string, readFile ReadFileFunc) Identity {
	candidates := selectCandidates(paths)
	if len(candidates) == 0 {
		return Identity{TSFID: "unknown"}
	}

	chosen := candidates[0]
	serial := ""
	for _, path := range candidates {
		content, err := readFile(path)
		if err != nil {
			continue
		}
		match := reSerial.FindStringSubmatch(string(content))
		if len(match) == 2 {
			serial = match[1]
			chosen = path
			break
		}
	}

	original := deriveOriginalName(filepath.Base(chosen))
	return Identity{
		TSFID:           serial + "|" + original,
		TSFOriginalName: original,
		Serial:          serial,
	}
}

func selectCandidates(paths []string) []string {
	matches1 := make([]string, 0)
	matches2 := make([]string, 0)
	for _, path := range paths {
		clean := filepath.ToSlash(filepath.Clean(path))
		if rePattern1.MatchString(clean) {
			matches1 = append(matches1, clean)
		}
		if rePattern2.MatchString(clean) {
			matches2 = append(matches2, clean)
		}
	}

	sort.Strings(matches1)
	sort.Strings(matches2)

	if len(matches1) > 0 {
		return matches1
	}
	return matches2
}

func deriveOriginalName(filename string) string {
	if strings.HasSuffix(filename, ".tgz.txt") || strings.HasSuffix(filename, ".tar.gz.txt") {
		return strings.TrimSuffix(filename, ".txt")
	}

	matches := findTSNameCandidates(filename)
	if len(matches) == 0 {
		return filename
	}

	best := matches[0]
	for _, m := range matches[1:] {
		if len(m) < len(best) {
			best = m
		}
	}
	return best
}

func findTSNameCandidates(filename string) []string {
	out := make([]string, 0)
	for i := 0; i < len(filename); i++ {
		if !isLetter(filename[i]) {
			continue
		}
		if i > 0 && isAlphaNum(filename[i-1]) {
			continue
		}
		m := reTSStart.FindString(filename[i:])
		if m != "" {
			out = append(out, m)
		}
	}
	return out
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isAlphaNum(b byte) bool {
	return isLetter(b) || (b >= '0' && b <= '9')
}
