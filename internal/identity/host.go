package identity

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const HostIdentityVersion = 1

type HostIdentity struct {
	Version   int       `json:"version"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func HostPathForCatalog(catalogPath string) string {
	catalogPath = strings.TrimSpace(catalogPath)
	if catalogPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(catalogPath), "host.json")
}

func LoadOrCreateHostForCatalog(catalogPath string) (HostIdentity, error) {
	return LoadOrCreateHost(HostPathForCatalog(catalogPath))
}

func LoadOrCreateHost(path string) (HostIdentity, error) {
	return loadOrCreateHost(path, time.Now, rand.Reader)
}

func LoadHost(path string) (HostIdentity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return HostIdentity{}, err
	}
	return decodeHost(data)
}

func loadOrCreateHost(path string, now func() time.Time, random io.Reader) (HostIdentity, error) {
	if strings.TrimSpace(path) == "" {
		return HostIdentity{}, errors.New("host identity path is required")
	}

	identity, err := LoadHost(path)
	if err == nil {
		return identity, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return HostIdentity{}, err
	}

	id, err := newHostID(random)
	if err != nil {
		return HostIdentity{}, err
	}
	identity = HostIdentity{
		Version:   HostIdentityVersion,
		ID:        id,
		CreatedAt: now().UTC(),
	}
	if err := identity.Validate(); err != nil {
		return HostIdentity{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return HostIdentity{}, err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return LoadHost(path)
	}
	if err != nil {
		return HostIdentity{}, err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(identity); err != nil {
		_ = file.Close()
		return HostIdentity{}, err
	}
	if err := file.Close(); err != nil {
		return HostIdentity{}, err
	}
	return identity, nil
}

func decodeHost(data []byte) (HostIdentity, error) {
	var identity HostIdentity
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&identity); err != nil {
		return HostIdentity{}, fmt.Errorf("decode host identity: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return HostIdentity{}, errors.New("decode host identity: multiple JSON values")
		}
		return HostIdentity{}, fmt.Errorf("decode host identity: %w", err)
	}
	if err := identity.Validate(); err != nil {
		return HostIdentity{}, err
	}
	return identity, nil
}

func (h HostIdentity) Validate() error {
	if h.Version != HostIdentityVersion {
		return fmt.Errorf("unsupported host identity version %d", h.Version)
	}
	if !validHostID(h.ID) {
		return fmt.Errorf("invalid host identity %q", h.ID)
	}
	if h.CreatedAt.IsZero() {
		return errors.New("host identity created_at is required")
	}
	return nil
}

func newHostID(random io.Reader) (string, error) {
	var raw [16]byte
	if _, err := io.ReadFull(random, raw[:]); err != nil {
		return "", fmt.Errorf("generate host identity: %w", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return "host:" + formatUUID(raw), nil
}

func formatUUID(raw [16]byte) string {
	encoded := make([]byte, 36)
	hex.Encode(encoded[0:8], raw[0:4])
	encoded[8] = '-'
	hex.Encode(encoded[9:13], raw[4:6])
	encoded[13] = '-'
	hex.Encode(encoded[14:18], raw[6:8])
	encoded[18] = '-'
	hex.Encode(encoded[19:23], raw[8:10])
	encoded[23] = '-'
	hex.Encode(encoded[24:36], raw[10:16])
	return string(encoded)
}

func validHostID(value string) bool {
	const prefix = "host:"
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	uuid := strings.TrimPrefix(value, prefix)
	if len(uuid) != 36 || uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		return false
	}
	for i, r := range uuid {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
