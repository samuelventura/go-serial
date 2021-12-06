//go:build windows

package serial

/*
// MSDN article on Serial Communications:
// http://msdn.microsoft.com/en-us/library/ff802693.aspx
// (alternative link) https://msdn.microsoft.com/en-us/library/ms810467.aspx
// Arduino Playground article on serial communication with Windows API:
// http://playground.arduino.cc/Interfacing/CPPWindows
*/

import (
	"io"
	"sync"
	"syscall"
)

type portDto struct {
	mu     sync.Mutex
	handle syscall.Handle
}

func GetPortsList() (list []string, err error) {
	list = []string{}
	subKey, err := syscall.UTF16PtrFromString("HARDWARE\\DEVICEMAP\\SERIALCOMM\\")
	if err != nil {
		return
	}

	var h syscall.Handle
	err = syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, subKey, 0, syscall.KEY_READ, &h)
	if err != nil {
		return
	}
	defer syscall.RegCloseKey(h)

	var valuesCount uint32
	err = syscall.RegQueryInfoKey(h, nil, nil, nil, nil, nil, nil, &valuesCount, nil, nil, nil, nil)
	if err != nil {
		return
	}
	list = make([]string, 0, valuesCount)
	for i := 0; i < cap(list); i++ {
		var data [1024]uint16
		dataSize := uint32(len(data))
		var name [1024]uint16
		nameSize := uint32(len(name))
		err = regEnumValue(h, uint32(i), &name[0], &nameSize, nil, nil, &data[0], &dataSize)
		if err != nil {
			return
		}
		list = append(list, syscall.UTF16ToString(data[:]))
	}
	return
}

func Open(portName string, mode *Mode) (port *portDto, err error) {
	portName = "\\\\.\\" + portName
	path, err := syscall.UTF16PtrFromString(portName)
	if err != nil {
		return
	}
	handle, err := syscall.CreateFile(
		path,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0, nil,
		syscall.OPEN_EXISTING,
		0,
		0)
	if err != nil {
		return
	}

	port = &portDto{
		handle: handle,
	}
	defer func() {
		if err != nil {
			port.Close()
		}
	}()

	params := dcb{}
	err = getCommState(handle, &params)
	if err != nil {
		return
	}
	params.BaudRate = uint32(mode.BaudRate)
	params.ByteSize = byte(mode.DataBits)
	params.StopBits = stopBitsMap[mode.StopBits]
	params.Parity = parityMap[mode.Parity]
	err = setCommState(handle, &params)
	if err != nil {
		return
	}

	err = port.SetReadTimeout(1000)
	if err != nil {
		return
	}

	return
}

func (port *portDto) SetReadTimeout(toms int) (err error) {
	rinter := uint32(0)
	rmult := uint32(0)
	rconst := uint32(0)
	if toms < 0 {
		rconst = 0xFFFFFFFF - 1
	}
	if toms > 0 {
		rconst = uint32(toms)
	}
	timeouts := &commTimeouts{
		ReadIntervalTimeout:           rinter,
		TimedReadtalTimeoutMultiplier: rmult,
		TimedReadtalTimeoutConstant:   rconst,
		WriteTotalTimeoutConstant:     0,
		WriteTotalTimeoutMultiplier:   0,
	}
	err = setCommTimeouts(port.handle, timeouts)
	err = tryConvertToEof(err)
	return
}

func (port *portDto) Read(p []byte) (n int, err error) {
	var count uint32
	err = syscall.ReadFile(port.handle, p, &count, nil)
	err = tryConvertToEof(err)
	n = int(count)
	return
}

func (port *portDto) Write(p []byte) (n int, err error) {
	var count uint32
	err = syscall.WriteFile(port.handle, p, &count, nil)
	err = tryConvertToEof(err)
	n = int(count)
	return
}

//mutex to allow safe multi close from go routines
func (port *portDto) Close() error {
	port.mu.Lock()
	defer func() {
		port.handle = 0
		port.mu.Unlock()
	}()
	if port.handle == 0 {
		return nil
	}
	return syscall.CloseHandle(port.handle)
}

func tryConvertToEof(in error) (out error) {
	out = in
	if in != nil {
		errno, ok := in.(syscall.Errno)
		//The handle is invalid
		if ok && 6 == uint(errno) {
			out = io.EOF
		}
	}
	return
}

type dcb struct {
	DCBlength uint32
	BaudRate  uint32

	// Flags field is a bitfield
	//  fBinary            :1
	//  fParity            :1
	//  fOutxCtsFlow       :1
	//  fOutxDsrFlow       :1
	//  fDtrControl        :2
	//  fDsrSensitivity    :1
	//  fTXContinueOnXoff  :1
	//  fOutX              :1
	//  fInX               :1
	//  fErrorChar         :1
	//  fNull              :1
	//  fRtsControl        :2
	//  fAbortOnError      :1
	//  fDummy2            :17
	Flags uint32

	wReserved  uint16
	XonLim     uint16
	XoffLim    uint16
	ByteSize   byte
	Parity     byte
	StopBits   byte
	XonChar    byte
	XoffChar   byte
	ErrorChar  byte
	EOFChar    byte
	EvtChar    byte
	wReserved1 uint16
}

type commTimeouts struct {
	ReadIntervalTimeout           uint32
	TimedReadtalTimeoutMultiplier uint32
	TimedReadtalTimeoutConstant   uint32
	WriteTotalTimeoutMultiplier   uint32
	WriteTotalTimeoutConstant     uint32
}

const (
	noParity   = 0
	oddParity  = 1
	evenParity = 2
)

var parityMap = map[Parity]byte{
	NoParity:   noParity,
	OddParity:  oddParity,
	EvenParity: evenParity,
}

const (
	oneStopBit   = 0
	one5StopBits = 1
	twoStopBits  = 2
)

var stopBitsMap = map[StopBits]byte{
	OneStopBit:           oneStopBit,
	OnePointFiveStopBits: one5StopBits,
	TwoStopBits:          twoStopBits,
}
