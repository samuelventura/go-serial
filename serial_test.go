package serial

import (
	"fmt"
	"io"
	"log"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/samuelventura/go-modbus"
)

func TestSerialEof(t *testing.T) {
	defer logPanic()
	log.SetFlags(log.Lmicroseconds)
	log.Println("PORT1", PORT1)
	log.Println("PORT2", PORT2)
	testSerialEof(t, PORT1)
	testSerialEof(t, PORT2)
}

func testSerialEof(t *testing.T, n string) {
	port := open(t, n)
	err := port.SetReadTimeout(100)
	iferr(t, err)
	err = port.Close()
	iferr(t, err)
	err = port.SetReadTimeout(100)
	if err != io.EOF {
		t.Fatalf("SetReadTimeout EOF not detected %v", err)
	}
	_, err = port.Read([]byte{0})
	if err != io.EOF {
		t.Fatalf("Read EOF not detected %v", err)
	}
	_, err = port.Write([]byte{0})
	if err != io.EOF {
		t.Fatalf("Write EOF not detected %v", err)
	}
}

func TestSerialTransport(t *testing.T) {
	defer logPanic()
	log.SetFlags(log.Lmicroseconds)
	//traceEnabled = true
	//modbus.EnableTrace(true)
	//EnableTrace(true)
	//test_modbus.EnableTrace(true)
	testSerialTrans(t, PORT1, PORT2, modbus.NewNopProtocol())
	testSerialTrans(t, PORT1, PORT2, modbus.NewRtuProtocol())
	testSerialTrans(t, PORT1, PORT2, modbus.NewTcpProtocol())
}

func testSerialTrans(t *testing.T, n string, m string, proto modbus.Protocol) {
	log.Println("Protocol", reflect.TypeOf(proto))
	port1 := open(t, n)
	port2 := open(t, m)
	done := make(chan bool)
	//linux previous slave listener wont close and
	//will receive new protocol data if not synced
	defer func() { <-done }()
	defer port1.Close()
	defer port2.Close()
	reader1 := NewTimedReader(port1)
	reader2 := NewTimedReader(port2)
	trans1 := modbus.NewIoTransport(reader1, port1)
	trans2 := modbus.NewIoTransport(reader2, port2)
	trans1.SetError(true)
	trans2.SetError(true)
	trans1.Discard(100)
	trans2.Discard(100)
	model := modbus.NewMapModel()
	master := modbus.NewMaster(proto, trans1, 400)
	go func() {
		defer func() { done <- true }()
		runSlave(proto, trans2, model, 100)
	}()
	testModelMaster(t, model, master)
}

func mode() *Mode {
	mode := &Mode{}
	//startech ftdi bit corruption at 57600 & 115200 on MacOS
	mode.BaudRate = 9600
	mode.DataBits = 8
	mode.Parity = NoParity
	mode.StopBits = OneStopBit
	return mode
}

func open(t *testing.T, n string) Port {
	mode := mode()
	port, err := Open(n, mode)
	iferr(t, err)
	return port
}

//MASTER////////////////////////////

func testModelMaster(t *testing.T, model modbus.Model, master modbus.CloseableMaster) {
	defer master.Close()
	var bools []bool
	var words []uint16
	var bool1 bool
	var word1 uint16
	var err error

	testWriteDos(t, model, master, 0, 0, randBools(modbus.MaxBools)...)
	testWriteWos(t, model, master, 0, 0, randWords(modbus.MaxWords)...)
	testReadDos(t, model, master, 0, 0, randBools(modbus.MaxBools)...)
	testReadWos(t, model, master, 0, 0, randWords(modbus.MaxWords)...)
	testReadDis(t, model, master, 0, 0, randBools(modbus.MaxBools)...)
	testReadWis(t, model, master, 0, 0, randWords(modbus.MaxWords)...)

	for k := 0; k < 10; k++ {
		testWriteDos(t, model, master, 0, 0, randBools(modbus.MaxBools-k)...)
		testWriteWos(t, model, master, 0, 0, randWords(modbus.MaxWords-k)...)
		testReadDos(t, model, master, 0, 0, randBools(modbus.MaxBools-k)...)
		testReadWos(t, model, master, 0, 0, randWords(modbus.MaxWords-k)...)
		testReadDis(t, model, master, 0, 0, randBools(modbus.MaxBools-k)...)
		testReadWis(t, model, master, 0, 0, randWords(modbus.MaxWords-k)...)
	}

	err = master.WriteDo(0xFF, 0xFFFF, false)
	if !strings.HasPrefix(err.Error(), fmt.Sprintf("ModbusException %02x", ^modbus.WriteDo05)) {
		t.Fatalf("Exception expected: %s", err.Error())
	}
	max := 0x10001
	start := time.Now().UnixNano()
	for k := 0; k < max; k++ {
		ifErrFatal(t, master.WriteDo(0, 0, false))
	}
	end := time.Now().UnixNano()
	totals := float64(end-start) / 1000000000.0
	unitms := float64(end-start) / float64(max) / 1000000.0
	log.Printf("Timed %fs %fms %d\n", totals, unitms, max)
	for ss := 0; ss < 0x1FF; ss += 50 {
		for aa := 0; aa < 0x1FFFF; aa += 10000 {
			s := byte(ss)
			a := uint16(aa)

			ifErrFatal(t, master.WriteDo(s, a, true))
			assertBoolsEqual(t, model.ReadDos(s, a, 1), []bool{true})
			bools, err = master.ReadDos(s, a, 1)
			assertBoolsEqualErr(t, err, bools, []bool{true})
			bool1, err = master.ReadDo(s, a)
			assertBoolEqualErr(t, err, bool1, true)
			ifErrFatal(t, master.WriteDo(s, a, false))
			assertBoolsEqual(t, model.ReadDos(s, a, 1), []bool{false})
			bools, err = master.ReadDos(s, a, 1)
			assertBoolsEqualErr(t, err, bools, []bool{false})
			bool1, err = master.ReadDo(s, a)
			assertBoolEqualErr(t, err, bool1, false)

			ifErrFatal(t, master.WriteWo(s, a, 0x37A5))
			assertWordsEqual(t, model.ReadWos(s, a, 1), []uint16{0x37A5})
			words, err = master.ReadWos(s, a, 1)
			assertWordsEqualErr(t, err, words, []uint16{0x37A5})
			word1, err = master.ReadWo(s, a)
			assertWordEqualErr(t, err, word1, 0x37A5)
			ifErrFatal(t, master.WriteWo(s, a, 0xC8F0))
			assertWordsEqual(t, model.ReadWos(s, a, 1), []uint16{0xC8F0})
			words, err = master.ReadWos(s, a, 1)
			assertWordsEqualErr(t, err, words, []uint16{0xC8F0})
			word1, err = master.ReadWo(s, a)
			assertWordEqualErr(t, err, word1, 0xC8F0)

			a += 1
			ifErrFatal(t, master.WriteDos(s, a, true, true))
			assertBoolsEqual(t, model.ReadDos(s, a, 2), []bool{true, true})
			ifErrFatal(t, master.WriteDos(s, a, false, true))
			assertBoolsEqual(t, model.ReadDos(s, a, 2), []bool{false, true})
			ifErrFatal(t, master.WriteDos(s, a, true, false))
			assertBoolsEqual(t, model.ReadDos(s, a, 2), []bool{true, false})
			ifErrFatal(t, master.WriteDos(s, a, false, false))
			assertBoolsEqual(t, model.ReadDos(s, a, 2), []bool{false, false})

			ifErrFatal(t, master.WriteWos(s, a, 0x37A5, 0xC8F0))
			assertWordsEqual(t, model.ReadWos(s, a, 2), []uint16{0x37A5, 0xC8F0})
			ifErrFatal(t, master.WriteWos(s, a, 0xC80F, 0x37A5))
			assertWordsEqual(t, model.ReadWos(s, a, 2), []uint16{0xC80F, 0x37A5})

			a += 2
			model.WriteDis(s, a, true, true)
			bools, err = master.ReadDis(s, a, 2)
			assertBoolsEqualErr(t, err, bools, []bool{true, true})
			bool1, err = master.ReadDi(s, a)
			assertBoolEqualErr(t, err, bool1, true)
			bool1, err = master.ReadDi(s, a+1)
			assertBoolEqualErr(t, err, bool1, true)
			model.WriteDis(s, a, false, true)
			bools, err = master.ReadDis(s, a, 2)
			assertBoolsEqualErr(t, err, bools, []bool{false, true})
			bool1, err = master.ReadDi(s, a)
			assertBoolEqualErr(t, err, bool1, false)
			bool1, err = master.ReadDi(s, a+1)
			assertBoolEqualErr(t, err, bool1, true)
			model.WriteDis(s, a, true, false)
			bools, err = master.ReadDis(s, a, 2)
			assertBoolsEqualErr(t, err, bools, []bool{true, false})
			bool1, err = master.ReadDi(s, a)
			assertBoolEqualErr(t, err, bool1, true)
			bool1, err = master.ReadDi(s, a+1)
			assertBoolEqualErr(t, err, bool1, false)
			model.WriteDis(s, a, false, false)
			bools, err = master.ReadDis(s, a, 2)
			assertBoolsEqualErr(t, err, bools, []bool{false, false})
			bool1, err = master.ReadDi(s, a)
			assertBoolEqualErr(t, err, bool1, false)
			bool1, err = master.ReadDi(s, a+1)
			assertBoolEqualErr(t, err, bool1, false)

			a += 2
			model.WriteWis(s, a, 0x37A5, 0xC8F0)
			words, err = master.ReadWis(s, a, 2)
			assertWordsEqualErr(t, err, words, []uint16{0x37A5, 0xC8F0})
			word1, err = master.ReadWi(s, a)
			assertWordEqualErr(t, err, word1, 0x37A5)
			word1, err = master.ReadWi(s, a+1)
			assertWordEqualErr(t, err, word1, 0xC8F0)
			model.WriteWis(s, a, 0xC80F, 0x375A)
			words, err = master.ReadWis(s, a, 2)
			assertWordsEqualErr(t, err, words, []uint16{0xC80F, 0x375A})
			word1, err = master.ReadWi(s, a)
			assertWordEqualErr(t, err, word1, 0xC80F)
			word1, err = master.ReadWi(s, a+1)
			assertWordEqualErr(t, err, word1, 0x375A)

			a += 2
			testBools(t, model, master, s, a, true, false, true, false, true, false, true, false, true, true, true, true, false, false, false, false, true)
			testBools(t, model, master, s, a, true, true, true, true, false, false, false, false, true, true, false, true, false, true, false, true, false)
			testWords(t, model, master, s, a, 0x0102, 0x0304, 0x0506, 0x0708, 0x09A0, 0x5A73, 0x000, 0xFFFF, 0xDE45, 0x98FE, 0x00FF, 0xFF00, 0x000, 0xFFFF)

			bools = randBools(20)
			words = randWords(20)
			for j := 1; j <= 20; j++ {
				testBools(t, model, master, s, a, bools[:j]...)
				testWords(t, model, master, s, a, words[:j]...)
			}
		}
	}
}

func testWriteDos(t *testing.T, model modbus.Model, master modbus.Master, s byte, a uint16, values ...bool) {
	ifErrFatal(t, master.WriteDos(s, a, values...))
	bools := model.ReadDos(s, a, uint16(len(values)))
	assertBoolsEqual(t, values, bools)
}

func testWriteWos(t *testing.T, model modbus.Model, master modbus.Master, s byte, a uint16, values ...uint16) {
	ifErrFatal(t, master.WriteWos(s, a, values...))
	words := model.ReadWos(s, a, uint16(len(values)))
	assertWordsEqual(t, values, words)
}

func testReadDos(t *testing.T, model modbus.Model, master modbus.Master, s byte, a uint16, values ...bool) {
	model.WriteDos(s, a, values...)
	bools, err := master.ReadDos(s, a, uint16(len(values)))
	assertBoolsEqualErr(t, err, values, bools)
}

func testReadWos(t *testing.T, model modbus.Model, master modbus.Master, s byte, a uint16, values ...uint16) {
	model.WriteWos(s, a, values...)
	words, err := master.ReadWos(s, a, uint16(len(values)))
	assertWordsEqualErr(t, err, values, words)
}

func testReadDis(t *testing.T, model modbus.Model, master modbus.Master, s byte, a uint16, values ...bool) {
	model.WriteDis(s, a, values...)
	bools, err := master.ReadDis(s, a, uint16(len(values)))
	assertBoolsEqualErr(t, err, values, bools)
}

func testReadWis(t *testing.T, model modbus.Model, master modbus.Master, s byte, a uint16, values ...uint16) {
	model.WriteWis(s, a, values...)
	words, err := master.ReadWis(s, a, uint16(len(values)))
	assertWordsEqualErr(t, err, values, words)
}

func testBools(t *testing.T, model modbus.Model, master modbus.Master, s byte, a uint16, values ...bool) {
	var err error
	var bools []bool
	model.WriteDis(s, a, values...)
	bools, err = master.ReadDis(s, a, uint16(len(values)))
	assertBoolsEqualErr(t, err, values, bools)
	model.WriteDos(s, a, values...)
	bools, err = master.ReadDos(s, a, uint16(len(values)))
	assertBoolsEqualErr(t, err, values, bools)
	master.WriteDos(s, a, values...)
	assertBoolsEqual(t, values, model.ReadDos(s, a, uint16(len(values))))
}

func testWords(t *testing.T, model modbus.Model, master modbus.Master, s byte, a uint16, values ...uint16) {
	var err error
	var words []uint16
	model.WriteWis(s, a, values...)
	words, err = master.ReadWis(s, a, uint16(len(values)))
	assertWordsEqualErr(t, err, values, words)
	model.WriteWos(s, a, values...)
	words, err = master.ReadWos(s, a, uint16(len(values)))
	assertWordsEqualErr(t, err, values, words)
	master.WriteWos(s, a, values...)
	assertWordsEqual(t, values, model.ReadWos(s, a, uint16(len(values))))
}

//SLAVE////////////////////////////

func runSlave(proto modbus.Protocol, trans modbus.Transport, exec modbus.Executor, toms int) {
	for {
		err := oneSlave(proto, trans, exec, toms)
		if err != nil {
			trace("oneSlave.error", toms, err)
		}
		if err == io.EOF {
			return
		}
	}
}

func oneSlave(proto modbus.Protocol, trans modbus.Transport, exec modbus.Executor, toms int) (err error) {
	//report error to transport
	//to discard on next interaction
	defer func() {
		trans.SetError(err != nil)
	}()
	trans.Discard(toms)
	for {
		ci, err := proto.Scan(trans, toms)
		if err != nil {
			return err
		}
		if ci.Slave == 0xFF && ci.Address == 0xFFFF {
			fbuf, buf := proto.MakeBuffers(3)
			buf[0] = ci.Slave
			buf[1] = ci.Code | 0x80
			buf[2] = ^ci.Code
			proto.WrapBuffer(fbuf, 3)
			trans.Write(fbuf)
			continue
		}
		_, buf, err := modbus.ApplyToExecutor(ci, proto, exec)
		if err != nil {
			return err
		}
		c, err := trans.Write(buf)
		if err != nil {
			return err
		}
		if c != len(buf) {
			return formatErr("Partial write %d of %d", c, len(buf))
		}
	}
}

//ASSERT////////////////////////////

func assertBoolEqualErr(t *testing.T, err error, a, b bool) {
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("Val mismatch %t %t", a, b)
	}
}

func assertBoolsEqualErr(t *testing.T, err error, a, b []bool) {
	if err != nil {
		t.Fatal(err)
	}
	assertBoolsEqual(t, a, b)
}

func assertBoolsEqual(t *testing.T, a, b []bool) {
	if len(a) != len(b) {
		t.Fatalf("Len mismatch %d %d", len(a), len(b))
	}
	for i, v := range a {
		if v != b[i] {
			t.Fatalf("Val mismatch at %d %t %t", i, v, b[i])
		}
	}
}

func assertWordEqualErr(t *testing.T, err error, a, b uint16) {
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("Val mismatch %04x %04x", a, b)
	}
}

func assertWordsEqualErr(t *testing.T, err error, a, b []uint16) {
	if err != nil {
		t.Fatal(err)
	}
	assertWordsEqual(t, a, b)
}

func assertWordsEqual(t *testing.T, a, b []uint16) {
	if len(a) != len(b) {
		t.Fatalf("Len mismatch %d %d", len(a), len(b))
	}
	for i, v := range a {
		if v != b[i] {
			t.Fatalf("Val mismatch at %d %04x %04x", i, v, b[i])
		}
	}
}

//RAND/////////////////////////////////////

func randBools(count int) (bools []bool) {
	bools = make([]bool, count)
	for i := range bools {
		bools[i] = randBool()
	}
	return
}

func randWords(count int) (words []uint16) {
	words = make([]uint16, count)
	for i := range words {
		words[i] = randWord()
	}
	return
}

func randBool() bool {
	return time.Now().UnixNano()%2 == 1
}

func randWord() uint16 {
	return uint16(time.Now().UnixNano() % 65536)
}

//TOOLS/////////////////////////////////////

func ifErrFatal(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func formatErr(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s %s", msg, string(debug.Stack()))
}

func logPanic() {
	if r := recover(); r != nil {
		log.Println(r, string(debug.Stack()))
	}
}

func iferr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
