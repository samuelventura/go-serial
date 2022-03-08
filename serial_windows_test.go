//go:build windows

package serial

// com0com-2.2.2.0-x64-fre-signed
// No need to check any setup flags

import (
	"log"
	"sort"
	"testing"
)

const (
	PORT1 = "COM98"
	PORT2 = "COM99"
)

func TestSerialFound(t *testing.T) {
	defer logPanic()
	log.SetFlags(log.Lmicroseconds)
	ports, err := GetPortsList()
	fatalIfError(t, err)
	sort.Strings(ports)
	np := len(ports)
	log.Println(np, ports)
	if np == 0 {
		t.Fatalf("Ports not found %d", np)
	}
	testSerialFound(t, ports, PORT1)
	testSerialFound(t, ports, PORT1)
}

func testSerialFound(t *testing.T, ports []string, name string) {
	n := sort.SearchStrings(ports, name)
	if n < 0 || n >= len(ports) {
		t.Fatalf("Port not found %s", name)
	}
}
