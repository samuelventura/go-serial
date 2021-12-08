package serial

import (
	"io"
	"log"
	"runtime/debug"
	"testing"

	"github.com/samuelventura/go-modbus"
	"github.com/samuelventura/go-modbus/spec"
)

func TestSerialEof(t *testing.T) {
	defer logPanic()
	//modbus.EnableTrace(true)
	log.SetFlags(log.Lmicroseconds)
	log.Println("PORT1", PORT1)
	log.Println("PORT2", PORT2)
	testSerialEof(t, PORT1)
	testSerialEof(t, PORT2)
}

func testSerialEof(t *testing.T, name string) {
	port := open(t, name)
	defer port.Close()
	err := port.SetReadTimeout(100)
	fatalIfError(t, err)
	err = port.Close()
	fatalIfError(t, err)
	err = port.SetReadTimeout(100)
	if err != io.EOF {
		t.Fatalf("setReadTimeout EOF not detected %v", err)
	}
	_, err = port.Read([]byte{0})
	if err != io.EOF {
		t.Fatalf("read EOF not detected %v", err)
	}
	_, err = port.Write([]byte{0})
	if err != io.EOF {
		t.Fatalf("write EOF not detected %v", err)
	}
}

func TestSerialTransport(t *testing.T) {
	defer logPanic()
	log.SetFlags(log.Lmicroseconds)
	setupMasterSlave(t, modbus.NewNopProtocol(), spec.ProtocolTest)
	setupMasterSlave(t, modbus.NewRtuProtocol(), spec.ProtocolTest)
	setupMasterSlave(t, modbus.NewTcpProtocol(), spec.ProtocolTest)
}

func setupMasterSlave(t *testing.T, proto modbus.Protocol, cb func(s *spec.SetupProtoTest)) {
	port1 := open(t, PORT1)
	port2 := open(t, PORT2)
	done := make(chan bool)
	//linux previous slave listener wont close and
	//will receive new protocol data if not synced
	defer func() { <-done }()
	defer port1.Close()
	defer port2.Close()
	setup := &spec.SetupProtoTest{}
	setup.T = t
	setup.Proto = proto
	reader1 := NewTimedReader(port1)
	reader2 := NewTimedReader(port2)
	trans1 := modbus.NewIoTransport(reader1, port1)
	trans2 := modbus.NewIoTransport(reader2, port2)
	trans1.DiscardOn()
	trans2.DiscardOn()
	//trans1.DiscardIf() //implicit
	trans2.DiscardIf() //slave side required?
	setup.Model = modbus.NewMapModel()
	exec := modbus.NewModelExecutor(setup.Model)
	execw := &spec.ExceptionExecutor{Exec: exec}
	setup.Master = modbus.NewMaster(proto, trans1, 400)
	defer setup.Master.Close()
	go func() {
		defer func() { done <- true }()
		modbus.RunSlave(proto, trans2, execw)
	}()
	cb(setup)
}

//startech ftdi bit corruption at 57600 & 115200 on MacOS
func mode() *Mode {
	mode := &Mode{}
	mode.BaudRate = 9600
	mode.DataBits = 8
	mode.Parity = NoParity
	mode.StopBits = OneStopBit
	return mode
}

func open(t *testing.T, name string) Port {
	mode := mode()
	port, err := Open(name, mode)
	fatalIfError(t, err)
	return port
}

//TOOLS/////////////////////////////////////

func logPanic() {
	if r := recover(); r != nil {
		log.Println(r, string(debug.Stack()))
	}
}

func fatalIfError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
