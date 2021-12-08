package serial

import "github.com/samuelventura/go-modbus"

func NewTimedReader(port Port) *portTimedReader {
	return &portTimedReader{port}
}

type portTimedReader struct {
	port Port
}

func (to portTimedReader) TimedRead(buf []byte) (c int, err error) {
	c = 0
	to.port.SetReadTimeout(modbus.ReadToMs)
	if err != nil {
		return
	}
	c, err = to.port.Read(buf)
	return
}

func NewSerialTransport(name string, mode *Mode) (trans modbus.Transport, err error) {
	port, err := Open(name, mode)
	if err != nil {
		return
	}
	trans = modbus.NewIoTransport(NewTimedReader(port), port)
	return
}
