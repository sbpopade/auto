// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qsfp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/platinasystems/go/internal/redis"
	"github.com/platinasystems/go/internal/machine"
)

type QsfpI2cGpioIo struct {
	init    int
	Present [2]uint16
}

var VdevIo [32]I2cDev

var VpageByKeyIo map[string]uint8

var qsfpIo = QsfpI2cGpioIo{1, [2]uint16{0xffff, 0xffff}}

var deviceVer byte
var porto int

func qsfpioTicker(c *Command) error {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-c.stop:
			return nil
		case <-t.C:
			if err := c.updateio(); err != nil {
				close(c.stop)
				return err
			}
		}
	}
}

func (c *Command) updateio() error {
	stopped := readStoppedio()
	if stopped == 1 {
		return nil
	}
	if qsfpIo.init == 1 {
		ss, _ := redis.Hget(machine.Name, "eeprom.DeviceVersion")
		_, _ = fmt.Sscan(ss, &deviceVer)
	}
	port := uint8(0)
	porto = 0
	if deviceVer == 0 || deviceVer == 0xff {
		porto = -1
	}
	for k, i := range VpageByKeyIo {
		for j := 1; j < 33; j++ {
			if strings.Contains(k, "port-"+strconv.Itoa(int(j)+porto)+".") {
				port = uint8(j) - 1
				break
			}
		}
		v := VdevIo[i].QsfpStatus(port)
		if v != c.lastsio[k] && v != "" {
			c.pub.Print(k, ": ", v)
			c.lastsio[k] = v
		}
	}
	return nil
}

func (h *I2cDev) QsfpStatus(port uint8) string {
	r := getRegs()
	var Present uint16
	if port == 0 || port == 16 {
		//initialize reset I2C GPIO
		if qsfpIo.init == 1 {
			VdevIo[0].QsfpInit(0xff, 0xff, 0x00, 0x00, 0xff, 0xff)
			VdevIo[1].QsfpInit(0xff, 0xff, 0x00, 0x00, 0xff, 0xff)
			VdevIo[2].QsfpInit(0xff, 0xff, 0x00, 0x00, 0xff, 0xff)
			VdevIo[3].QsfpInit(0xff, 0xff, 0x00, 0x00, 0xff, 0xff)
			VdevIo[4].QsfpInit(0xff, 0xff, 0x00, 0x00, 0x00, 0x00)
			VdevIo[5].QsfpInit(0xff, 0xff, 0x00, 0x00, 0x00, 0x00)
			VdevIo[6].QsfpInit(0x0, 0x0, 0x00, 0x00, 0x0, 0x0)
			VdevIo[7].QsfpInit(0x0, 0x0, 0x00, 0x00, 0x0, 0x0)
			qsfpIo.init = 0
		}

		r.Input[0].get(h)
		r.Input[1].get(h)
		closeMuxio(h)
		err := DoI2cRpcio()
		if err != nil {
			return ""
		}
		p := uint16(sio[1].D[0])
		p += uint16(sio[3].D[0]) << 8
		if port == 0 && qsfpIo.Present[0] != p {
			//Take port out of reset and LP mode if qsfp is installed
			VdevIo[6].QsfpReset((p ^ qsfpIo.Present[0]), p^0xffff)
			VdevIo[4].QsfpLpMode((p ^ qsfpIo.Present[0]), p)
			qsfpIo.Present[0] = p

			//send to qspi.go
			SendPresentToQsfp()
		} else if port == 16 && qsfpIo.Present[1] != p {
			//Take port out of reset and LP mode if qsfp is installed
			VdevIo[7].QsfpReset((p ^ qsfpIo.Present[1]), p^0xffff)
			VdevIo[5].QsfpLpMode((p ^ qsfpIo.Present[1]), p)
			qsfpIo.Present[1] = p

			//send to qspi.go
			SendPresentToQsfp()
		}
	}

	if port < 16 {
		Present = qsfpIo.Present[0]
	} else {
		Present = qsfpIo.Present[1]
	}

	//swap upper/lower ports to match front panel numbering
	if (port % 2) == 0 {
		port++
	} else {
		port--
	}

	pmask := uint16(1) << (port % 16)
	if (Present&pmask)>>(port%16) == 1 {
		return "empty"
	}
	return "installed"
}

func (h *I2cDev) QsfpReset(ports uint16, reset uint16) {

	//if module was removed or inserted into a port, set reset line accordingly
	r := getRegs()
	if (ports & 0xff) != 0 {
		r.Output[0].get(h)
		closeMuxio(h)
		DoI2cRpcio()
		v := uint8((sio[1].D[0] & uint8((ports&0xff)^0xff)) | uint8((ports&reset)&0xff))
		r.Output[0].set(h, v)
	}
	if (ports & 0xff00) != 0 {
		r.Output[1].get(h)
		closeMuxio(h)
		DoI2cRpcio()
		v := uint8((sio[1].D[0] & uint8(((ports&0xff00)>>8)^0xff)) | uint8(((ports&reset)&0xff00)>>8))
		r.Output[1].set(h, v)
	}
	closeMuxio(h)
	DoI2cRpcio()
	return
}

func (h *I2cDev) QsfpLpMode(ports uint16, reset uint16) {

	//if module was removed or inserted into a port, set reset line accordingly
	r := getRegs()
	if (ports & 0xff) != 0 {
		r.Output[0].get(h)
		closeMuxio(h)
		DoI2cRpcio()
		v := uint8((sio[1].D[0] & uint8((ports&0xff)^0xff)) | uint8((ports&reset)&0xff))
		r.Output[0].set(h, v)
	}
	if (ports & 0xff00) != 0 {
		r.Output[1].get(h)
		closeMuxio(h)
		DoI2cRpcio()
		v := uint8((sio[1].D[0] & uint8(((ports&0xff00)>>8)^0xff)) | uint8(((ports&reset)&0xff00)>>8))
		r.Output[1].set(h, v)
	}
	closeMuxio(h)
	DoI2cRpcio()
	return
}

func (h *I2cDev) QsfpInit(out0 byte, out1 byte, pol0 byte, pol1 byte, conf0 byte, conf1 byte) {
	//all ports default in reset
	r := getRegs()
	r.Output[0].set(h, out0)
	r.Output[1].set(h, out1)
	closeMuxio(h)
	DoI2cRpcio()
	r.Polarity[0].set(h, pol0)
	r.Polarity[1].set(h, pol1)
	closeMuxio(h)
	DoI2cRpcio()
	r.Config[0].set(h, conf0)
	r.Config[1].set(h, conf1)
	closeMuxio(h)
	DoI2cRpcio()
	return
}

func SendPresentToQsfp() {
	latestPresent = qsfpIo.Present
}
