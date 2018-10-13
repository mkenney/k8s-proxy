package proxy

import (
	"testing"
)

// fake test
func TestProxy(t *testing.T) {
	proxy := &Proxy{}
	if nil == proxy {
		t.Errorf("Expected value, received nil")
	}
}
