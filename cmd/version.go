package cmd

import (
	"fmt"
	"regexp"
)

// versionPattern matches a full Go release number such as 1.21.0.
var versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// validateVersionArg checks that args holds exactly one version of the form
// MAJOR.MINOR.PATCH, without the "go" prefix.
func validateVersionArg(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expect one argument")
	}
	if !versionPattern.MatchString(args[0]) {
		return fmt.Errorf("invalid version format. Please enter a valid version (e.g. %s)", MinVersion)
	}
	return nil
}
