package federationhttp

import "net/url"

func validatePeerURL(rawURL string) (*url.URL, error) {
	return ValidatePeerURL(rawURL)
}
