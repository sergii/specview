package federationhttp

import (
	"net"
	"testing"
)

func TestDefaultAddressIsLoopbackOnly(t *testing.T) {
	host, port, err := net.SplitHostPort(DefaultAddress)
	if err != nil {
		t.Fatalf("DefaultAddress %q is invalid: %v", DefaultAddress, err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		t.Fatalf("DefaultAddress host %q must be loopback", host)
	}
	if port == "" {
		t.Fatal("DefaultAddress must include a port")
	}
}
