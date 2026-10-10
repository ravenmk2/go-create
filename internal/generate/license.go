package generate

import (
	"embed"
	"fmt"
	"strconv"
	"strings"
)

//go:embed licenses
var licensesFS embed.FS

// LicenseIDs returns the supported license identifiers.
func LicenseIDs() []string {
	return []string{"mit", "apache-2.0", "gpl-3.0"}
}

// ValidLicense reports whether id is a supported license (an empty string means no license is generated).
func ValidLicense(id string) bool {
	if id == "" || id == "none" {
		return true
	}
	for _, v := range LicenseIDs() {
		if v == id {
			return true
		}
	}
	return false
}

// LicenseText returns the full license text with copyright information substituted.
func LicenseText(id, holder string, year int) (string, error) {
	data, err := licensesFS.ReadFile("licenses/" + id + ".txt")
	if err != nil {
		return "", fmt.Errorf("unknown license %q", id)
	}
	s := string(data)
	if id == "mit" {
		s = strings.Replace(s, "<year>", strconv.Itoa(year), 1)
		s = strings.Replace(s, "<copyright holders>", holder, 1)
	}
	return s, nil
}
