// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

// Package w83795d provides access to the H/W Monitor chip

package w83795d

import (
	"net/rpc"
	"time"
	"unsafe"

	"github.com/platinasystems/go/internal/i2c"
	"github.com/platinasystems/go/internal/log"
)

const MAXOPS = 30

type I struct {
	InUse     bool
	RW        i2c.RW
	RegOffset uint8
	BusSize   i2c.SMBusSize
	Data      [34]byte
	Bus       int
	Addr      int
	Delay     int
}
type R struct {
	D [34]byte
	E error
}

type I2cReq int

var b = [34]byte{0}
var i = I{false, i2c.RW(0), 0, 0, b, 0, 0, 0}
var j [MAXOPS]I
var r = R{b, nil}
var s [MAXOPS]R
var x int

var dummy byte
var regsPointer = unsafe.Pointer(&dummy)
var regsAddr = uintptr(unsafe.Pointer(&dummy))

var clientA *rpc.Client
var dialed int = 0

func getRegsBank0() *regsBank0 {
	clearJ()
	return (*regsBank0)(regsPointer)
}

func getRegsBank2() *regsBank2 {
	clearJ()
	return (*regsBank2)(regsPointer)
}
func (r *reg8) offset() uint8   { return uint8(uintptr(unsafe.Pointer(r)) - regsAddr) }
func (r *reg16) offset() uint8  { return uint8(uintptr(unsafe.Pointer(r)) - regsAddr) }
func (r *reg16r) offset() uint8 { return uint8(uintptr(unsafe.Pointer(r)) - regsAddr) }

func closeMux(h *I2cDev) {
	var data = [34]byte{0, 0, 0, 0}

	data[0] = byte(0)
	j[x] = I{true, i2c.Write, 0, i2c.ByteData, data, h.MuxBus, h.MuxAddr, 0}
	x++

}

func (r *reg8) get(h *I2cDev) {
	var data = [34]byte{0, 0, 0, 0}

	data[0] = byte(h.MuxValue)
	j[x] = I{true, i2c.Write, 0, i2c.ByteData, data, h.MuxBus, h.MuxAddr, 0}
	x++
	j[x] = I{true, i2c.Read, r.offset(), i2c.ByteData, data, h.Bus, h.Addr, 0}
	x++
}

func (r *reg16) get(h *I2cDev) {
	var data = [34]byte{0, 0, 0, 0}

	data[0] = byte(h.MuxValue)
	j[x] = I{true, i2c.Write, 0, i2c.ByteData, data, h.MuxBus, h.MuxAddr, 0}
	x++
	j[x] = I{true, i2c.Read, r.offset(), i2c.WordData, data, h.Bus, h.Addr, 0}
	x++
}

func (r *reg16r) get(h *I2cDev) {
	var data = [34]byte{0, 0, 0, 0}

	data[0] = byte(h.MuxValue)
	j[x] = I{true, i2c.Write, 0, i2c.ByteData, data, h.MuxBus, h.MuxAddr, 0}
	x++
	j[x] = I{true, i2c.Read, r.offset(), i2c.WordData, data, h.Bus, h.Addr, 0}
	x++
}

func (r *reg8) set(h *I2cDev, v uint8) {
	var data = [34]byte{0, 0, 0, 0}

	data[0] = byte(h.MuxValue)
	j[x] = I{true, i2c.Write, 0, i2c.ByteData, data, h.MuxBus, h.MuxAddr, 0}
	x++
	data[0] = v
	j[x] = I{true, i2c.Write, r.offset(), i2c.ByteData, data, h.Bus, h.Addr, 0}
	x++
}

func (r *reg16) set(h *I2cDev, v uint16) {
	var data = [34]byte{0, 0, 0, 0}

	data[0] = byte(h.MuxValue)
	j[x] = I{true, i2c.Write, 0, i2c.ByteData, data, h.MuxBus, h.MuxAddr, 0}
	x++
	data[0] = uint8(v >> 8)
	data[1] = uint8(v)
	j[x] = I{true, i2c.Write, r.offset(), i2c.WordData, data, h.Bus, h.Addr, 0}
	x++
}

func (r *reg16r) set(h *I2cDev, v uint16) {
	var data = [34]byte{0, 0, 0, 0}

	data[0] = byte(h.MuxValue)
	j[x] = I{true, i2c.Write, 0, i2c.ByteData, data, h.MuxBus, h.MuxAddr, 0}
	x++
	data[1] = uint8(v >> 8)
	data[0] = uint8(v)
	j[x] = I{true, i2c.Write, r.offset(), i2c.WordData, data, h.Bus, h.Addr, 0}
	x++
}

func clearJ() {
	x = 0
	for k := 0; k < MAXOPS; k++ {
		j[k] = i
	}
}

func readStopped() byte {
	var data = [34]byte{0, 0, 0, 0}
	j[0] = I{true, i2c.Write, 0, i2c.ByteData, data, int(0x98), int(0), 0}
	err := DoI2cRpc()
	if err != nil {
		return 1
	}
	return s[0].D[0]
}

func DoI2cRpc() error {
	if dialed == 0 {
		client, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1233")
		if err != nil {
			log.Print("dialing:", err)
			return err
		}
		clientA = client
		dialed = 1
		time.Sleep(time.Millisecond * time.Duration(50))
	}
	err := clientA.Call("I2cReq.ReadWrite", &j, &s)
	if err != nil {
		log.Print("i2cReq error:", err)
		return err
	}
	clearJ()
	return nil
}
