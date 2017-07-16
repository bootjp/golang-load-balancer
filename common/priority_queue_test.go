package common

import (
	"testing"
)

func TestNextRoundRobin(t *testing.T) {
	backendA := Backend{Ip: "0.0.0.0", Port: "8081", ActiveConnections: 0}
	backendB := Backend{Ip: "0.0.0.0", Port: "8082", ActiveConnections: 0}
	backends := Backends{0, []Backend{backendA, backendB}}

	firstBackend := NextRoundRobin(&backends)
	secondBackend := NextRoundRobin(&backends)
	thirdBackend := NextRoundRobin(&backends)

	if firstBackend != backendA {
		t.Errorf("Wrong order of NextRoundRobin(), expected %v, got: %v", backendA, firstBackend)
	}
	if secondBackend != backendB {
		t.Errorf("Wrong order of NextRoundRobin(), expected %v, got: %v", backendB, firstBackend)
	}
	if thirdBackend != backendA {
		t.Errorf("Wrong order of NextRoundRobin(), expected %v, got: %v", backendA, firstBackend)
	}
}