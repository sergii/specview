package federationpeers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)

func ResolveCredentialHeaders(credentials *CredentialRef) (http.Header, error) {
	if credentials == nil {
		return nil, nil
	}
	if err := validateCredentials(*credentials); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(credentials.Headers))
	for name := range credentials.Headers {
		names = append(names, name)
	}
	sort.Strings(names)

	headers := make(http.Header, len(names))
	for _, headerName := range names {
		envName := credentials.Headers[headerName]
		value, ok := os.LookupEnv(envName)
		if !ok || strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("federation credential environment variable %s is not set", envName)
		}
		headers.Set(headerName, value)
	}
	if len(headers) == 0 {
		return nil, errors.New("federation credential headers are empty")
	}
	return headers, nil
}
