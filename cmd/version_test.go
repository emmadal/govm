package cmd

import (
	"testing"
)

func TestValidateVersionArg(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"valid", []string{"1.21.0"}, false},
		{"valid two digit patch", []string{"1.25.12"}, false},
		{"no argument", nil, true},
		{"two arguments", []string{"1.21.0", "1.22.0"}, true},
		{"go prefix", []string{"go1.21.0"}, true},
		{"not a version", []string{"abcdef"}, true},
		{"two segments", []string{"1.21"}, true},
		{"four segments", []string{"1.21.0.1"}, true},
		{"release candidate", []string{"1.22rc1"}, true},
		{"empty", []string{""}, true},
		{"path traversal", []string{"../../1.2.3"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateVersionArg(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("validateVersionArg(%q) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

// Regression: these invocations used to panic with an index out of range.
func TestCommands_BadArgumentsReturnErrorWithoutPanic(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	for _, args := range [][]string{
		{"install"},
		{"use"},
		{"rm"},
		{"install", "abcdef"},
		{"install", "1.21"},
		{"use", "abcdef"},
		{"rm", "abcdef"},
		{"install", "1.20.0"}, // below MinVersion
	} {
		t.Run(joinArgs(args), func(t *testing.T) {
			initCmd.SetArgs(args)
			if err := initCmd.Execute(); err == nil {
				t.Errorf("govm %v: expected an error", args)
			}
		})
	}
}

func joinArgs(args []string) string {
	name := ""
	for _, arg := range args {
		name += arg + " "
	}
	return name
}
