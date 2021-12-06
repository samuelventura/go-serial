//go:build linux || darwin || freebsd || openbsd

package serial

const (
	PORT1 = "/tmp/tty.fake.master" //1
	PORT2 = "/tmp/tty.fake.slave"  //2
)
