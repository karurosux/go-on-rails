package run

import (
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestCheckPortFree(t *testing.T) {
	// Grab an ephemeral port, hold it, and expect CheckPortFree to refuse it.
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)

	err = CheckPortFree(port)
	if err == nil {
		t.Fatalf("expected an error for taken port %s", port)
	}
	if !strings.Contains(err.Error(), port) {
		t.Errorf("error should name the port %s: %v", port, err)
	}

	// Released, the same port should pass.
	l.Close()
	if err := CheckPortFree(port); err != nil {
		t.Errorf("expected free port %s to pass: %v", port, err)
	}
}
