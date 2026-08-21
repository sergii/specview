//go:build !linux && !darwin

package hoststate

func (CodexScanner) Scan() ([]Observation, error) {
	return nil, nil
}
