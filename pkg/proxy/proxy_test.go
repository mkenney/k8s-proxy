package proxy

import (
	"testing"
)

func TestProxy(t *testing.T) {
	proxy := &Proxy{}
	if nil == proxy {
		t.Errorf("Expected value, received nil")
	}
}
