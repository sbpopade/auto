// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qsfp

import (
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	redigo "github.com/garyburd/redigo/redis"
	"github.com/platinasystems/go/goes/cmd"
	"github.com/platinasystems/go/goes/lang"
	"github.com/platinasystems/go/internal/log"
	"github.com/platinasystems/go/internal/machine"
	"github.com/platinasystems/go/internal/redis"
	"github.com/platinasystems/go/internal/redis/publisher"
	"github.com/platinasystems/go/vnet/platforms/mk1"
)

var Vdev [32]I2cDev

type Command struct {
	Init func()
	init sync.Once

	stop    chan struct{}
	pub     *publisher.Publisher
	last    map[string]float64
	lasts   map[string]string
	lastu   map[string]uint8
	lastsio map[string]string
}

type I2cDev struct {
	Bus       int
	Addr      int
	MuxBus    int
	MuxAddr   int
	MuxValue  int
	MuxBus2   int
	MuxAddr2  int
	MuxValue2 int
}

var portLpage0 [32]lpage0
var portUpage3 [32]upage3
var portIsCopper [32]bool
var maxTemp float64
var maxTempPort string
var bmcIpv6LinkLocalRedis string

var VpageByKey map[string]uint8

var latestPresent = [2]uint16{0xffff, 0xffff}
var present = [2]uint16{0xffff, 0xffff}

func (*Command) String() string { return "qsfp" }

func (*Command) Usage() string { return "qsfp" }

func (*Command) Apropos() lang.Alt {
	return lang.Alt{
		lang.EnUS: "qsfp monitoring daemon, publishes to redis",
	}
}

func (c *Command) Close() error {
	close(c.stop)
	return nil
}

func (*Command) Kind() cmd.Kind { return cmd.Daemon }

func (c *Command) Main(...string) error {
	var err error
	var si syscall.Sysinfo_t

	if c.Init != nil {
		c.init.Do(c.Init)
	}

	if err = redis.IsReady(); err != nil {
		return err
	}

	for i := 0; i < 32; i++ {
		portIsCopper[i] = true
	}
	maxTemp = 0
	maxTempPort = "-1"
	c.stop = make(chan struct{})
	c.last = make(map[string]float64)
	c.lasts = make(map[string]string)
	c.lastu = make(map[string]uint8)
	c.lastsio = make(map[string]string)

	if c.pub, err = publisher.New(); err != nil {
		return err
	}
	if err = syscall.Sysinfo(&si); err != nil {
		return err
	}

	go qsfpioTicker(c)
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	tm := time.NewTicker(500 * time.Millisecond)
	defer tm.Stop()
	for {
		select {
		case <-c.stop:
			return nil
		case <-t.C:
			if err = c.updateMonitor(); err != nil {
				close(c.stop)
				return err
			}
		case <-tm.C:
			if err = c.updatePresence(); err != nil {
				close(c.stop)
				return err
			}
		}
	}

	return nil
}

func (c *Command) updatePresence() error {
	stopped := readStopped()
	if stopped == 1 {
		return nil
	}
	ready, err := redis.Hget(machine.Name, "vnet.ready")
	if err != nil || ready == "false" {
		return nil
	}

	var buffPresent = [2]uint16{0xffff, 0xffff}

	for j := 0; j < 2; j++ {
		//when qsfp is installed or removed from a port
		buffPresent[j] = latestPresent[j]
		if present[j] != buffPresent[j] {
			for i := 0; i < 16; i++ {
				if (1<<uint(i))&(buffPresent[j]^present[j]) != 0 {
					//physical to logical port translation
					lp := i + j*16
					if (lp % 2) == 0 {
						lp += 2
					}
					lp += porto
					var typeString string
					if ((1 << uint(i)) & (buffPresent[j] ^ 0xffff)) != 0 {
						//when qsfp is installed publish static data
						k := "port-" + strconv.Itoa(lp) + ".qsfp.compliance"
						v := Vdev[i+j*16].Compliance()
						var portConfig string

						//identify copper vs optic and set media and speed
						media, err := redis.Hget(machine.Name, "vnet."+mk1.IfnameOf(lp, (1+porto))+".media")
						if err != nil {
							log.Print("qsfp hget error:", err)
						}
						speed, err := redis.Hget(machine.Name, "vnet."+mk1.IfnameOf(lp, (1+porto))+".speed")
						if err != nil {
							log.Print("qsfp hget error:", err)
						}
						speed = strings.ToLower(speed)
						if strings.Contains(v, "-CR") {
							portIsCopper[i+j*16] = true
							if media != "copper" {
								ret, err := redis.Hset(machine.Name, "vnet."+mk1.IfnameOf(lp, (1+porto))+".media", "copper")
								if err != nil || ret != 1 {
									log.Print("qsfp hset error:", err, " ", ret)
								} else {
									portConfig += "copper "
								}
							}
							if speed == "100g" {
								ret, err := redis.Hset(machine.Name, "vnet."+mk1.IfnameOf(lp, (1+porto))+".fec", "cl91")
								if err != nil || ret != 1 {
									log.Print("qsfp hset error:", err, " ", ret)
								} else {
									portConfig += "cl91 "
								}
							}
						} else if strings.Contains(v, "40G") {
							portIsCopper[i+j*16] = false
							Vdev[i+j*16].TxDisableSet(false, lp)
							if media != "fiber" {
								ret, err := redis.Hset(machine.Name, "vnet."+mk1.IfnameOf(lp, (1+porto))+".media", "fiber")
								if err != nil || ret != 1 {
									log.Print("qsfp hset error:", err, " ", ret)
								} else {
									portConfig += "fiber "
								}
							}
							if speed != "40g" {
								ret, err := redis.Hset(machine.Name, "vnet."+mk1.IfnameOf(lp, (1+porto))+".speed", "40g")
								if err != nil || ret != 1 {
									log.Print("qsfp hset error:", err, " ", ret)
								} else {
									portConfig += "40g fixed speed"
								}
							}
						} else {
							portIsCopper[i+j*16] = false
							Vdev[i+j*16].TxDisableSet(false, lp)
							if media != "fiber" {
								ret, err := redis.Hset(machine.Name, "vnet."+mk1.IfnameOf(lp, (1+porto))+".media", "fiber")
								if err != nil || ret != 1 {
									log.Print("qsfp hset error:", err, " ", ret)
								} else {
									portConfig += "fiber "
								}
							}
							if speed != "100g" {
								ret, err := redis.Hset(machine.Name, "vnet."+mk1.IfnameOf(lp, (1+porto))+".speed", "100g")
								if err != nil || ret != 1 {
									log.Print("qsfp hset error:", err, " ", ret)
								} else {
									portConfig += "100g fixed speed"
								}
							}
						}
						typeString += strings.Trim(v, " ") + ", "
						if v != c.lasts[k] {
							c.pub.Print(k, ": ", v)
							c.lasts[k] = v
						}

						k = "port-" + strconv.Itoa(lp) + ".qsfp.vendor"
						v = Vdev[i+j*16].Vendor()
						typeString += strings.Trim(v, " ") + ", "
						if v != c.lasts[k] {
							c.pub.Print(k, ": ", v)
							c.lasts[k] = v
						}
						k = "port-" + strconv.Itoa(lp) + ".qsfp.partnumber"
						v = Vdev[i+j*16].PN()
						typeString += strings.Trim(v, " ") + ", "
						if v != c.lasts[k] {
							c.pub.Print(k, ": ", v)
							c.lasts[k] = v
						}
						k = "port-" + strconv.Itoa(lp) + ".qsfp.serialnumber"
						v = Vdev[i+j*16].SN()
						typeString += strings.Trim(v, " ")
						if v != c.lasts[k] {
							c.pub.Print(k, ": ", v)
						}
						if !portIsCopper[i+j*16] {
							// optics need delay from power on to make thresholds readable
							time.Sleep(10 * time.Millisecond)
							// get monitoring thresholds if qsfp is not a cable
							Vdev[i+j*16].StaticBlocks(i + j*16)
							v = Temp(portUpage3[i+j*16].tempHighAlarm)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.temperature.highAlarmThreshold.units.C"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Temp(portUpage3[i+j*16].tempLowAlarm)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.temperature.lowAlarmThreshold.units.C"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Temp(portUpage3[i+j*16].tempHighWarning)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.temperature.highWarnThreshold.units.C"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Temp(portUpage3[i+j*16].tempLowWarning)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.temperature.lowWarnThreshold.units.C"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Voltage(portUpage3[i+j*16].vccHighAlarm)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.vcc.highAlarmThreshold.units.V"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Voltage(portUpage3[i+j*16].vccLowAlarm)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.vcc.lowAlarmThreshold.units.V"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Voltage(portUpage3[i+j*16].vccHighWarning)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.vcc.highWarnThreshold.units.V"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Voltage(portUpage3[i+j*16].vccLowWarning)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.vcc.lowWarnThreshold.units.V"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Power(portUpage3[i+j*16].rxPowerHighAlarm)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.rx.power.highAlarmThreshold.units.mW"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Power(portUpage3[i+j*16].rxPowerLowAlarm)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.rx.power.lowAlarmThreshold.units.mW"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Power(portUpage3[i+j*16].rxPowerHighWarning)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.rx.power.highWarnThreshold.units.mW"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Power(portUpage3[i+j*16].rxPowerLowWarning)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.rx.power.lowWarnThreshold.units.mW"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Power(portUpage3[i+j*16].txPowerHighAlarm)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.tx.power.highAlarmThreshold.units.mW"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Power(portUpage3[i+j*16].txPowerLowAlarm)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.tx.power.lowAlarmThreshold.units.mW"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Power(portUpage3[i+j*16].txPowerHighWarning)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.tx.power.highWarnThreshold.units.mW"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Power(portUpage3[i+j*16].txPowerLowWarning)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.tx.power.lowWarnThreshold.units.mW"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = TxBias(portUpage3[i+j*16].txBiasHighAlarm)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.tx.biasHighAlarmThreshold.units.mA"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = TxBias(portUpage3[i+j*16].txBiasLowAlarm)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.tx.biasLowAlarmThreshold.units.mA"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = TxBias(portUpage3[i+j*16].txBiasHighWarning)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.tx.biasHighWarnThreshold.units.mA"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = TxBias(portUpage3[i+j*16].txBiasLowWarning)
							k = "port-" + strconv.Itoa(lp) + ".qsfp.tx.biasLowWarnThreshold.units.mA"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
							v = Vdev[i+j*16].TxDisableGet()
							k = "port-" + strconv.Itoa(lp) + ".qsfp.txDisable"
							if v != c.lasts[k] {
								c.pub.Print(k, ": ", v)
								c.lasts[k] = v
							}
						}
						log.Print("QSFP detected in port ", lp, ": ", typeString)
						if portConfig != "" {
							log.Print("Port ", lp, " setting changed to ", portConfig)
						}
					} else {
						//when qsfp is removed, delete associated fields
						for _, v := range redisFields {
							k := "port-" + strconv.Itoa(lp) + "." + v
							c.pub.Print("delete: ", k)
							c.lasts[k] = ""
						}
						log.Print("QSFP removed from port ", lp)
						portIsCopper[i+j*16] = true
						if maxTempPort == strconv.Itoa(lp) {
							maxTempPort = "-1"
						}
					}
				}
			}
		}
		present[j] = buffPresent[j]
	}

	return nil
}

func (c *Command) updateMonitor() error {
	stopped := readStopped()
	if stopped == 1 {
		return nil
	}
	ready, err := redis.Hget(machine.Name, "vnet.ready")
	if err != nil || ready == "false" {
		return nil
	}

	for i := 0; i < 32; i++ {
		//publish dynamic monitoring data
		var port int
		if (i % 2) == 0 {
			port = i + 2
		} else {
			port = i
		}
		port += porto
		if present[i/16]&(1<<uint(i%16)) == 0 {
			if !portIsCopper[i] {
				// get monitoring data only if qsfp is present and not a cable
				if Vdev[i].DataReady() {
					Vdev[i].DynamicBlocks(i)
					k := "port-" + strconv.Itoa(port) + ".qsfp.temperature.units.C"
					v := CheckTemp(portLpage0[i].freeMonTemp, strconv.Itoa(port))
					if v != c.lasts[k] {
						c.pub.Print(k, ": ", v)
						c.lasts[k] = v
					}
					k = "port-" + strconv.Itoa(port) + ".qsfp.vcc.units.V"
					v = Voltage(portLpage0[i].freeMonVoltage)
					if v != c.lasts[k] {
						c.pub.Print(k, ": ", v)
						c.lasts[k] = v
					}
					va := LanePower(portLpage0[i].rxPower)
					for x := 0; x < 4; x++ {
						k = "port-" + strconv.Itoa(port) + ".qsfp.rx" + strconv.Itoa(x+1) + ".power.units.mW"
						if va[x] != c.lasts[k] {
							c.pub.Print(k, ": ", va[x])
							c.lasts[k] = va[x]
						}
					}
					va = LanePower(portLpage0[i].txPower)
					for x := 0; x < 4; x++ {
						k = "port-" + strconv.Itoa(port) + ".qsfp.tx" + strconv.Itoa(x+1) + ".power.units.mW"
						if va[x] != c.lasts[k] {
							c.pub.Print(k, ": ", va[x])
							c.lasts[k] = va[x]
						}
					}
					va = LanesTxBias(portLpage0[i].txBias)
					for x := 0; x < 4; x++ {
						k = "port-" + strconv.Itoa(port) + ".qsfp.tx" + strconv.Itoa(x+1) + ".bias.units.mA"
						if va[x] != c.lasts[k] {
							c.pub.Print(k, ": ", va[x])
							c.lasts[k] = va[x]
						}
					}
					vs := ChannelAlarms(portLpage0[i].channelStatusInterrupt, portLpage0[i].channelMonitorInterruptFlags)
					for x := 0; x < 4; x++ {
						k = "port-" + strconv.Itoa(port) + ".qsfp.rx" + strconv.Itoa(x+1) + ".alarms"
						if vs[x] != c.lasts[k] {
							c.pub.Print(k, ": ", vs[x])
							c.lasts[k] = vs[x]
						}
					}
					for x := 4; x < 8; x++ {
						k = "port-" + strconv.Itoa(port) + ".qsfp.tx" + strconv.Itoa(x-3) + ".alarms"
						if vs[x] != c.lasts[k] {
							c.pub.Print(k, ": ", vs[x])
							c.lasts[k] = vs[x]
						}
					}
					vs[0] = FreeSideAlarms(portLpage0[i].freeMonitorInterruptFlags)
					k = "port-" + strconv.Itoa(port) + ".qsfp.alarms"
					if vs[0] != c.lasts[k] {
						c.pub.Print(k, ": ", vs[0])
						c.lasts[k] = vs[0]
					}
				}
			}
		}

	}

	return nil
}

func (h *I2cDev) DataReady() bool {
	var t bool
	r := getRegsLpage0()

	r.status.get(h)
	closeMux(h)
	DoI2cRpc()

	if (s[2].D[1] & 0x1) == 1 {
		t = false
	} else {
		t = true
	}

	return t
}

func (h *I2cDev) Compliance() string {
	r := getRegsUpage0()

	r.SpecCompliance.get(h)
	closeMux(h)
	DoI2cRpc()
	cp := s[2].D[0]

	r.ExtSpecCompliance.get(h)
	closeMux(h)
	DoI2cRpc()
	ecp := s[2].D[0]

	var t string
	if (cp & 0x80) != 0x80 {
		t = specComplianceValues[cp]
	} else {
		t = extSpecComplianceValues[ecp]
	}
	return t
}

func (h *I2cDev) TxDisableSet(t bool, p int) {
	var b, e bool

	r := getRegsLpage0()
	r.txDisable.get(h)
	closeMux(h)
	DoI2cRpc()
	d := s[2].D[0]

	if d == 0x0 {
		b = false
	} else {
		b = true
	}

	if t {
		r.txDisable.set(h, 0xf)
		closeMux(h)
		DoI2cRpc()
	} else {
		r.txDisable.set(h, 0x0)
		closeMux(h)
		DoI2cRpc()
	}

	r.txDisable.get(h)
	closeMux(h)
	DoI2cRpc()
	d = s[2].D[0]

	if d == 0x0 {
		e = false
	} else {
		e = true
	}

	if e == t {
		log.Print("Port ", strconv.Itoa(p), " Tx Disable set to ", t, ". Previous value = ", b)
	} else {
		log.Print("Warning: Port ", strconv.Itoa(p), "cannot set Tx Disable to ", t, ". Current vlaue = ", e)
	}
	return
}

func (h *I2cDev) TxDisableGet() string {
	r := getRegsLpage0()
	var v string

	r.txDisable.get(h)
	closeMux(h)
	DoI2cRpc()
	d := s[2].D[0]

	if d == 0 {
		v = "false"
	} else {
		v = "true"
	}

	return v
}

func (h *I2cDev) Vendor() string {
	r := getRegsUpage0()
	r.VendorName.get(h, 16)
	closeMux(h)
	DoI2cRpc()
	t := string(s[2].D[1:16])

	return t
}

func (h *I2cDev) PN() string {
	r := getRegsUpage0()
	r.VendorPN.get(h, 16)
	closeMux(h)
	DoI2cRpc()
	t := string(s[2].D[1:16])

	return t
}

func (h *I2cDev) SN() string {
	r := getRegsUpage0()
	r.VendorSN.get(h, 16)
	closeMux(h)
	DoI2cRpc()
	t := string(s[2].D[1:16])

	return t
}

func LanePower(t [8]byte) [4]string {
	var v [4]string
	var u uint16
	for i := 0; i < 4; i++ {
		u = uint16(t[i*2])<<8 + uint16(t[i*2+1])
		v[i] = strconv.FormatFloat(float64(u)*0.0001, 'f', 3, 64)
	}
	return v
}

func Power(t uint16) string {
	v := strconv.FormatFloat(float64(t)*0.0001, 'f', 3, 64)
	return v
}

func LanesTxBias(t [8]byte) [4]string {
	var v [4]string
	var u uint16
	for i := 0; i < 4; i++ {
		u = uint16(t[i*2])<<8 + uint16(t[i*2+1])
		v[i] = strconv.FormatFloat(float64(u)*0.002, 'f', 3, 64)
	}
	return v
}

func TxBias(t uint16) string {
	v := strconv.FormatFloat(float64(t)*0.002, 'f', 3, 64)
	return v
}

func FreeSideAlarms(t [3]byte) string {
	var v string
	if ((1 << 4) & t[0]) != 0 {
		v += "TempLowWarn,"
	}
	if ((1 << 5) & t[0]) != 0 {
		v += "TempHighWarn,"
	}
	if ((1 << 6) & t[0]) != 0 {
		v += "TempLowAlarm,"
	}
	if ((1 << 7) & t[0]) != 0 {
		v += "TempHighAlarm,"
	}
	if ((1 << 4) & t[1]) != 0 {
		v += "VccLowWarn,"
	}
	if ((1 << 5) & t[1]) != 0 {
		v += "VccHighWarn,"
	}
	if ((1 << 6) & t[1]) != 0 {
		v += "VccLowAlarm,"
	}
	if ((1 << 7) & t[1]) != 0 {
		v += "VccHighAlarm,"
	}
	if v == "" {
		v = "none"
	}
	v = strings.Trim(v, ",")
	return v
}

func ChannelAlarms(t [3]byte, w [6]byte) [8]string {
	var v [8]string
	for i := uint(0); i < 4; i++ {
		if ((1 << i) & t[0]) != 0 {
			v[i] += "RxLos,"
		}
		if ((1 << (i + 4)) & t[0]) != 0 {
			v[i+4] += "TxLos,"
		}
		if ((1 << i) & t[1]) != 0 {
			v[i+4] += "TxFault,"
		}
		if ((1 << (i + 4)) & t[1]) != 0 {
			v[i+4] += "TxEqFault,"
		}
		if ((1 << i) & t[2]) != 0 {
			v[i] += "RxCdrLol,"
		}
		if ((1 << (i + 4)) & t[2]) != 0 {
			v[i+4] += "TxCdrLol,"
		}
		if ((1 << (4 - (i%2)*4)) & w[i/2]) != 0 {
			v[i] += "PowerLowWarn,"
		}
		if ((1 << (5 - (i%2)*4)) & w[i/2]) != 0 {
			v[i] += "PowerHighWarn,"
		}
		if ((1 << (6 - (i%2)*4)) & w[i/2]) != 0 {
			v[i] += "PowerLowAlarm,"
		}
		if ((1 << (7 - (i%2)*4)) & w[i/2]) != 0 {
			v[i] += "PowerHighAlarm,"
		}
		if ((1 << (4 - (i%2)*4)) & w[i/2+2]) != 0 {
			v[i+4] += "BiasLowWarn,"
		}
		if ((1 << (5 - (i%2)*4)) & w[i/2+2]) != 0 {
			v[i+4] += "BiasHighWarn,"
		}
		if ((1 << (6 - (i%2)*4)) & w[i/2+2]) != 0 {
			v[i+4] += "BiasLowAlarm,"
		}
		if ((1 << (7 - (i%2)*4)) & w[i/2+2]) != 0 {
			v[i+4] += "BiasHighAlarm,"
		}
		if ((1 << (4 - (i%2)*4)) & w[i/2+4]) != 0 {
			v[i+4] += "PowerLowWarn,"
		}
		if ((1 << (5 - (i%2)*4)) & w[i/2+4]) != 0 {
			v[i+4] += "PowerHighWarn,"
		}
		if ((1 << (6 - (i%2)*4)) & w[i/2+4]) != 0 {
			v[i+4] += "PowerLowAlarm,"
		}
		if ((1 << (7 - (i%2)*4)) & w[i/2+4]) != 0 {
			v[i+4] += "PowerHighAlarm,"
		}
		if v[i] == "" {
			v[i] = "none"
		}
		if v[i+4] == "" {
			v[i+4] = "none"
		}
		v[i] = strings.Trim(v[i], ",")
		v[i+4] = strings.Trim(v[i+4], ",")
	}
	return v
}

func CheckTemp(t uint16, port string) string {
	var u float64
	var v string
	update := false

	if (t & 0x8000) != 0 {
		u = float64((t^0xffff)+1) / 256 * (-1)
	} else {
		u = float64(t) / 256
	}
	v = strconv.FormatFloat(u, 'f', 1, 64)

	if maxTempPort == "-1" {
		maxTemp = u
		maxTempPort = port
		update = true
	} else if maxTempPort == port {
		maxTemp = u
		update = true
	} else if u > maxTemp {
		maxTemp = u
		maxTempPort = port
		update = true
	}

	if update {
		if bmcIpv6LinkLocalRedis == "" {
			m, err := redis.Hget(machine.Name, "eeprom.BaseEthernetAddress")
			if err == nil {
				o := strings.Split(m, ":")
				b, _ := hex.DecodeString(o[0])
				b[0] = b[0] ^ byte(2)
				o[0] = hex.EncodeToString(b)
				bmcIpv6LinkLocalRedis = "[fe80::" + o[0] + o[1] + ":" + o[2] + "ff:fe" + o[3] + ":" + o[4] + o[5] + "%eth0]:6379"
			}
		}
		if bmcIpv6LinkLocalRedis != "" {
			d, err := redigo.Dial("tcp", bmcIpv6LinkLocalRedis)
			if err != nil {
				log.Print(err)
			} else {
				_, err = d.Do("HSET", machine.Name+"-bmc", "qsfp.temp.units.C", v)
				if err != nil {
					d.Do("HSET", "platina", "qsfp.temp.units.C", v) // to support old bmc builds
				}
				d.Close()
			}
		}

	}

	return v
}

func Temp(t uint16) string {
	var u float64
	var v string
	if (t & 0x8000) != 0 {
		u = float64((t^0xffff)+1) / 256 * (-1)
	} else {
		u = float64(t) / 256
	}
	v = strconv.FormatFloat(u, 'f', 1, 64)
	return v
}

func Voltage(t uint16) string {
	var u float64
	var v string
	u = float64(t) * 0.0001
	v = strconv.FormatFloat(u, 'f', 3, 64)
	return v
}
func (h *I2cDev) DynamicBlocks(port int) {
	r := getRegsLpage0()
	rb := getBlocks()

	r.pageSelect.set(h, 0)
	rb.lpage0b.get(h, 32)
	closeMux(h)
	DoI2cRpc()

	portLpage0[port].id = s[5].D[1]
	portLpage0[port].status = uint16(s[5].D[2]) + uint16(s[5].D[3])<<8
	copy(portLpage0[port].channelStatusInterrupt[:], s[5].D[4:7])
	copy(portLpage0[port].freeMonitorInterruptFlags[:], s[5].D[7:10])
	copy(portLpage0[port].channelMonitorInterruptFlags[:], s[5].D[10:16])
	portLpage0[port].freeMonTemp = uint16(s[5].D[24]) + uint16(s[5].D[23])<<8
	portLpage0[port].freeMonVoltage = uint16(s[5].D[28]) + uint16(s[5].D[27])<<8

	r.pageSelect.set(h, 0)
	rb.lpage1b.get(h, 32)
	closeMux(h)
	DoI2cRpc()

	copy(portLpage0[port].rxPower[:], s[5].D[3:11])
	copy(portLpage0[port].txBias[:], s[5].D[11:19])
	copy(portLpage0[port].txPower[:], s[5].D[19:27])
}

func (h *I2cDev) StaticBlocks(port int) {
	if !portIsCopper[port] {
		r := getRegsLpage0()
		rb := getBlocks()

		r.pageSelect.set(h, 3)
		rb.upage0b.get(h, 32)
		closeMux(h)
		DoI2cRpc()
		portUpage3[port].tempHighAlarm = (uint16(s[5].D[1]) << 8) + uint16(s[5].D[2])
		portUpage3[port].tempLowAlarm = (uint16(s[5].D[3]) << 8) + uint16(s[5].D[4])
		portUpage3[port].tempHighWarning = (uint16(s[5].D[5]) << 8) + uint16(s[5].D[6])
		portUpage3[port].tempLowWarning = (uint16(s[5].D[7]) << 8) + uint16(s[5].D[8])
		portUpage3[port].vccHighAlarm = (uint16(s[5].D[17]) << 8) + uint16(s[5].D[18])
		portUpage3[port].vccLowAlarm = (uint16(s[5].D[19]) << 8) + uint16(s[5].D[20])
		portUpage3[port].vccHighWarning = (uint16(s[5].D[21]) << 8) + uint16(s[5].D[22])
		portUpage3[port].vccLowWarning = (uint16(s[5].D[23]) << 8) + uint16(s[5].D[24])

		r.pageSelect.set(h, 3)
		rb.upage1b.get(h, 32)
		closeMux(h)
		DoI2cRpc()
		portUpage3[port].rxPowerHighAlarm = (uint16(s[5].D[17]) << 8) + uint16(s[5].D[18])
		portUpage3[port].rxPowerLowAlarm = (uint16(s[5].D[19]) << 8) + uint16(s[5].D[20])
		portUpage3[port].rxPowerHighWarning = (uint16(s[5].D[21]) << 8) + uint16(s[5].D[22])
		portUpage3[port].rxPowerLowWarning = (uint16(s[5].D[23]) << 8) + uint16(s[5].D[24])
		portUpage3[port].txBiasHighAlarm = (uint16(s[5].D[25]) << 8) + uint16(s[5].D[26])
		portUpage3[port].txBiasLowAlarm = (uint16(s[5].D[27]) << 8) + uint16(s[5].D[28])
		portUpage3[port].txBiasHighWarning = (uint16(s[5].D[29]) << 8) + uint16(s[5].D[30])
		portUpage3[port].txBiasLowWarning = (uint16(s[5].D[31]) << 8) + uint16(s[5].D[32])

		r.pageSelect.set(h, 3)
		rb.upage2b.get(h, 32)
		closeMux(h)
		DoI2cRpc()
		portUpage3[port].txPowerHighAlarm = (uint16(s[5].D[1]) << 8) + uint16(s[5].D[2])
		portUpage3[port].txPowerLowAlarm = (uint16(s[5].D[3]) << 8) + uint16(s[5].D[4])
		portUpage3[port].txPowerHighWarning = (uint16(s[5].D[5]) << 8) + uint16(s[5].D[6])
		portUpage3[port].txPowerLowWarning = (uint16(s[5].D[7]) << 8) + uint16(s[5].D[8])

		r = getRegsLpage0()
		r.pageSelect.set(h, 0)
		closeMux(h)
		DoI2cRpc()
	}
	return
}
