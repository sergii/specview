package identity

import (
	"fmt"
	"strings"
)

// ValidateHostID validates the public opaque Host identity format.
func ValidateHostID(value string) error {
	value = strings.TrimSpace(value)
	if !validHostID(value) {
		return fmt.Errorf("invalid host identity %q", value)
	}
	return nil
}
