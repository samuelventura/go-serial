package serial

//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output zsyscall_windows.go syscall_windows.go

//only expected errors are timeout and eof
//despite different, closed will be reported as EOF
//SetReadTimeout, Read, and Write must detect EOF
type Port interface {
	SetReadTimeout(toms int) error
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
}

type Mode struct {
	BaudRate int      // platform dependant
	DataBits int      // 7 or 8
	Parity   Parity   // None, Odd and Even
	StopBits StopBits // 1, 1.5, 2
}

type Parity int

const (
	NoParity Parity = iota
	OddParity
	EvenParity
)

type StopBits int

const (
	OneStopBit StopBits = iota
	OnePointFiveStopBits
	TwoStopBits
)
