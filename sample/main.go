package main

import (
	"log"
	"time"

	"github.com/samuelventura/go-modbus"
	"github.com/samuelventura/go-serial"
)

//(cd sample; go run .)
func main() {
	log.SetFlags(log.Lmicroseconds)
	mode := &serial.Mode{}
	mode.BaudRate = 9600
	mode.DataBits = 8
	mode.Parity = serial.NoParity
	mode.StopBits = serial.OneStopBit
	trans, err := serial.NewSerialTransport("/dev/ttyUSB0", mode)
	if err != nil {
		log.Fatal(err)
	}
	defer trans.Close()
	trans.DiscardOn()
	modbus.EnableTrace(true)
	master := modbus.NewRtuMaster(trans, 400)
	for {
		err = master.WriteDo(1, 4105, true)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
		err = master.WriteDo(1, 4105, false)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
