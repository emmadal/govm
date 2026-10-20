package pkg

import (
	"cmp"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// versionRe matches Go release numbers: 1.22, 1.22.3, 1.22rc1, 1.22beta2.
var versionRe = regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:(beta|rc)(\d+))?$`)

// NormalizeVersion trims an optional "go" prefix and validates the result.
// "go1.22.0" and "1.22.0" both return "1.22.0".
func NormalizeVersion(input string) (string, error) {
	v := strings.TrimPrefix(strings.TrimSpace(input), "go")
	if !versionRe.MatchString(v) {
		return "", fmt.Errorf("invalid version %q: expected a form like 1.22.0, 1.22 or 1.22rc1", input)
	}
	return v, nil
}

type parsedVersion struct {
	major, minor, patch int
	stage               int // 0 beta, 1 rc, 2 final
	pre                 int
}

func parseVersion(v string) (parsedVersion, bool) {
	m := versionRe.FindStringSubmatch(strings.TrimPrefix(v, "go"))
	if m == nil {
		return parsedVersion{}, false
	}
	atoi := func(s string) int {
		n, _ := strconv.Atoi(s)
		return n
	}
	p := parsedVersion{major: atoi(m[1]), minor: atoi(m[2]), patch: atoi(m[3]), stage: 2, pre: atoi(m[5])}
	switch m[4] {
	case "beta":
		p.stage = 0
	case "rc":
		p.stage = 1
	}
	return p, true
}

// CompareVersions returns -1, 0 or +1 depending on whether a is older,
// equal to, or newer than b. Pre-releases sort before the final release.
// Unparseable versions sort first.
func CompareVersions(a, b string) int {
	pa, okA := parseVersion(a)
	pb, okB := parseVersion(b)
	switch {
	case !okA && !okB:
		return strings.Compare(a, b)
	case !okA:
		return -1
	case !okB:
		return 1
	}
	return cmp.Or(
		cmp.Compare(pa.major, pb.major),
		cmp.Compare(pa.minor, pb.minor),
		cmp.Compare(pa.patch, pb.patch),
		cmp.Compare(pa.stage, pb.stage),
		cmp.Compare(pa.pre, pb.pre),
	)
}

// isMinorLine reports whether v names only a minor line such as "1.22".
func isMinorLine(v string) bool {
	p, ok := parseVersion(v)
	return ok && v == fmt.Sprintf("%d.%d", p.major, p.minor)
}

// inMinorLine reports whether v is a final patch release of the line (e.g. 1.22.3 in 1.22).
func inMinorLine(v, line string) bool {
	p, ok := parseVersion(v)
	return ok && p.stage == 2 && strings.HasPrefix(v, line+".")
}
