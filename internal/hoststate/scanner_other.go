//go:build !linux && !darwin

package hoststate

func (CodexScanner) Scan() ([]Observation, error) {
	return nil, nil
}

func diagnoseCodex() ([]ScanDiagnostic, error) {
	return nil, nil
}
