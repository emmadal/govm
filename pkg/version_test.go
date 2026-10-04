package pkg

import "testing"

func TestNormalizeVersion(t *testing.T) {
	valid := map[string]string{
		"1.22.0":      "1.22.0",
		"go1.22.0":    "1.22.0",
		" 1.22 ":      "1.22",
		"1.23rc1":     "1.23rc1",
		"go1.21beta2": "1.21beta2",
	}
	for in, want := range valid {
		got, err := NormalizeVersion(in)
		if err != nil || got != want {
			t.Errorf("NormalizeVersion(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "go", "1", "1.22.0.1", "latest", "1.22-rc1", "v1.22.0", "1.22.0; rm -rf /"} {
		if _, err := NormalizeVersion(in); err == nil {
			t.Errorf("NormalizeVersion(%q) succeeded, want error", in)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.22.0", "1.22.0", 0},
		{"1.22.1", "1.22.0", 1},
		{"1.9.0", "1.10.0", -1},
		{"1.22rc1", "1.22.0", -1},
		{"1.22beta1", "1.22rc1", -1},
		{"1.22rc2", "1.22rc1", 1},
		{"1.20", "1.20rc3", 1},
		{"1.20.1", "1.20", 1},
		{"go1.21.0", "1.21.0", 0},
		{"garbage", "1.0", -1},
	}
	for _, tt := range tests {
		if got := CompareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestMinorLine(t *testing.T) {
	if !isMinorLine("1.22") || isMinorLine("1.22.0") || isMinorLine("1.22rc1") {
		t.Error("isMinorLine misclassified a version")
	}
	if !inMinorLine("1.22.3", "1.22") || inMinorLine("1.22rc1", "1.22") || inMinorLine("1.2.3", "1.22") || inMinorLine("1.221.0", "1.22") {
		t.Error("inMinorLine misclassified a version")
	}
}
