//go:build linux || darwin || freebsd || openbsd

package serial

const (
	PORT1 = "/tmp/tty.master" //1
	PORT2 = "/tmp/tty.slave"  //2
)
