//go:build !linux && !darwin

package hoststate

func diagnoseClaude() ([]ScanDiagnostic, error) {
	return nil, nil
}
