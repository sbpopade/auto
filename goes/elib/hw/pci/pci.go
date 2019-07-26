// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Generic devices on PCI bus.
package pci

import (
	"github.com/platinasystems/go/elib/hw"

	"fmt"
	"sync"
	"unsafe"
)

type U8 hw.U8
type U16 hw.U16
type U32 hw.U32

func (r *U8) offset() uint  { return uint((*hw.U8)(r).Offset()) }
func (r *U16) offset() uint { return uint((*hw.U16)(r).Offset()) }
func (r *U32) offset() uint { return uint((*hw.U32)(r).Offset()) }

func (r *U8) Get(d *Device) uint8      { return d.ReadConfigUint8(r.offset()) }
func (r *U8) Set(d *Device, v uint8)   { d.WriteConfigUint8(r.offset(), v) }
func (r *U16) Get(d *Device) uint16    { return d.ReadConfigUint16(r.offset()) }
func (r *U16) Set(d *Device, v uint16) { d.WriteConfigUint16(r.offset(), v) }
func (r *U32) Get(d *Device) uint32    { return d.ReadConfigUint32(r.offset()) }
func (r *U32) Set(d *Device, v uint32) { d.WriteConfigUint32(r.offset(), v) }

func (r *U8) GetRaw(d *Device) uint8      { return d.ReadRawConfigUint8(r.offset()) }
func (r *U8) SetRaw(d *Device, v uint8)   { d.WriteRawConfigUint8(r.offset(), v) }
func (r *U16) GetRaw(d *Device) uint16    { return d.ReadRawConfigUint16(r.offset()) }
func (r *U16) SetRaw(d *Device, v uint16) { d.WriteRawConfigUint16(r.offset(), v) }
func (r *U32) GetRaw(d *Device) uint32    { return d.ReadRawConfigUint32(r.offset()) }
func (r *U32) SetRaw(d *Device, v uint32) { d.WriteRawConfigUint32(r.offset(), v) }

func (d *Device) getRegs(o uint) unsafe.Pointer { return unsafe.Pointer(hw.BaseAddress + uintptr(o)) }
func (d *Device) GetConfig() *ConfigHeader      { return (*ConfigHeader)(d.getRegs(0)) }

// Under PCI, each device has 256 bytes of configuration address space,
// of which the first 64 bytes are standardized as follows:
type ConfigHeader struct {
	DeviceID
	Command
	Status

	Revision U8

	// Distinguishes programming interface for device.
	// For example, different standards for USB controllers.
	SoftwareInterface

	DeviceClass

	CacheSize    uint8
	LatencyTimer uint8

	// If bit 7 of this register is set, the device has multiple functions;
	// otherwise, it is a single function device.
	Tp uint8

	Bist uint8
}

type HeaderType uint8

func (c ConfigHeader) Type() HeaderType {
	return HeaderType(c.Tp &^ (1 << 7))
}

const (
	Normal HeaderType = iota
	Bridge
	CardBus
)

type SoftwareInterface U8

func (x SoftwareInterface) String() string {
	return fmt.Sprintf("0x%02x", uint8(x))
}

type Command U16

func (c *Command) Get(d *Device) Command    { return Command((*U16)(c).Get(d)) }
func (c *Command) Set(d *Device, v Command) { (*U16)(c).Set(d, uint16(v)) }

const (
	IOEnable Command = 1 << iota
	MemoryEnable
	BusMasterEnable
	SpecialCycles
	WriteInvalidate
	VgaPaletteSnoop
	Parity
	AddressDataStepping
	SERR
	BackToBackWrite
	INTxEmulationDisable
)

type Status U16

// Device/vendor ID from PCI config space.
type VendorID U16
type VendorDeviceID U16

func (r *DeviceID) Get(d *Device) (i DeviceID) {
	i.Vendor = r.Vendor.Get(d)
	i.Device = r.Device.Get(d)
	return
}
func (r *VendorID) Get(d *Device) VendorID             { return VendorID((*U16)(r).Get(d)) }
func (r *VendorDeviceID) Get(d *Device) VendorDeviceID { return VendorDeviceID((*U16)(r).Get(d)) }

func (d VendorDeviceID) String() string { return fmt.Sprintf("0x%04x", uint16(d)) }

// Vendor/Device pair
type DeviceID struct {
	Vendor VendorID
	Device VendorDeviceID
}

func (d *Device) VendorID() VendorID       { return d.ID.Vendor }
func (d *Device) DeviceID() VendorDeviceID { return d.ID.Device }

type BaseAddressReg U32

func (b BaseAddressReg) IsMem() bool {
	return b&(1<<0) == 0
}

func (b BaseAddressReg) Addr() uint32 {
	return uint32(b &^ 0xf)
}

func (b BaseAddressReg) Valid() bool {
	return b.Addr() != 0
}

func (b BaseAddressReg) String() string {
	if b == 0 {
		return "{}"
	}
	x := uint32(b)
	tp := "mem"
	loc := ""
	if !b.IsMem() {
		tp = "i/o"
	} else {
		switch (x >> 1) & 3 {
		case 0:
			loc = "32-bit "
		case 1:
			loc = "< 1M "
		case 2:
			loc = "64-bit "
		case 3:
			loc = "unknown "
		}
		if x&(1<<3) != 0 {
			loc += "prefetchable "
		}
	}
	return fmt.Sprintf("{%s: %s0x%08x}", tp, loc, b.Addr())
}

/* Header type 0 (normal devices) */
type DeviceConfig struct {
	ConfigHeader

	// Base addresses specify locations in memory or I/O space.
	// Decoded size can be determined by writing a value of 0xffffffff to the register, and reading it back.
	// Only 1 bits are decoded.
	BaseAddressRegs [6]BaseAddressReg

	CardBusCIS U32

	SubID DeviceID

	RomAddress U32

	// Config space offset of start of capability list.
	CapabilityOffset U8
	_                [7]U8

	InterruptLine U8
	InterruptPin  U8
	MinGrant      U8
	MaxLatency    U8
}

func (d *Device) GetDeviceConfig() *DeviceConfig { return (*DeviceConfig)(d.getRegs(0)) }

type Capability U8

const (
	PowerManagement Capability = iota + 1
	AGP
	VitalProductData
	SlotIdentification
	MSI
	CompactPCIHotSwap
	PCIX
	HyperTransport
	VendorSpecific
	DebugPort
	CompactPciCentralControl
	PCIHotPlugController
	SSVID
	AGP3
	SecureDevice
	PCIE
	MSIX
	SATA
	AdvancedFeatures
)

// Common header for capabilities.
type CapabilityHeader struct {
	Capability

	// Pointer to next capability header
	NextCapabilityHeader U8
}

type ExtCapability U16

const (
	AdvancedErrorReporting ExtCapability = iota + 1
	VirtualChannel
	DeviceSerialNumber
	PowerBudgeting
	RootComplexLinkDeclaration
	RootComplexInternalLinkControl
	RootComplexEventCollector
	MultiFunctionVC
	VirtualChannel9
	RootComplexRB
	ExtVendorSpecific
	ConfigAccess
	AccessControlServices
	AlternateRoutingID
	AddressTranslationServices
	SingleRootIOVirtualization
	MultiRootIOVirtualization
	Multicast
	PageRequestInterface
	ReservedAMD
	ResizableBAR
	DynamicPowerAllocation
	TPHRequester
	LatencyToleranceReporting
	SecondaryPCIeCapability
	ProtocolMultiplexing
	ProcessAddressSpaceID
)

// Common header for extended capabilities.
type ExtCapabilityHeader struct {
	ExtCapability

	// [15:4] next pointer
	// [3:0] version
	VersionAndNextOffset U16
}

type BusAddress struct {
	Domain        uint16
	Bus, Slot, Fn uint8
}

func (a BusAddress) String() string {
	return fmt.Sprintf("%04x:%02x:%02x.%01x", a.Domain, a.Bus, a.Slot, a.Fn)
}

type Resource struct {
	Index      uint32 // index of BAR
	Base, Size uint64
	Mem        []byte
}

func (r Resource) String() string {
	return fmt.Sprintf("{%d: 0x%x-0x%x}", r.Index, r.Base, r.Base+r.Size-1)
}

func (d *Device) String() string {
	return fmt.Sprintf("%s %v %v", &d.Addr, d.VendorID(), d.DeviceID())
}

type Device struct {
	ID        DeviceID
	Addr      BusAddress
	Resources []Resource
	Driver
	DriverDevice
	BusDevice
}

// Set bus master in pci command register.
// Otherwise no love with device dma or msi interrupts.
func (d *Device) SetMaster(enable bool) {
	c := d.GetConfig()
	v := c.Command.Get(d)
	if enable {
		v |= BusMasterEnable
	} else {
		v &^= BusMasterEnable
	}
	c.Command.Set(d, v)
}

// Things a driver must do.
type Driver interface {
	// Device matches registered devices for this driver.
	NewDevice(d BusDevice) (i DriverDevice, err error)
}

// This a device handled by driver must do.
type DriverDevice interface {
	Init() (err error)
	Exit() (err error)
	Interrupt()
}

type busCommon struct {
	registeredDevs []BusDevice
}

func (b *busCommon) getBusCommon() *busCommon { return b }

type Bus interface {
	getBusCommon() *busCommon
	NewDevice() BusDevice
	Validate() error
}

// Things a bus driver device must do.
type BusDevice interface {
	ConfigRw(offset, vʹ, nBytes uint, isWrite bool) (v uint)
	GetDevice() *Device
	Open() error
	Close() error
	MapResource(index uint) (res uintptr, err error)
	UnmapResource(index uint) (err error)
	InterruptEnable(UseMsi bool) (err error)
}

var (
	driversMutex sync.Mutex
	drivers      map[DeviceID]Driver = make(map[DeviceID]Driver)
)

func setDriver(v Driver, id DeviceID) (err error) {
	driversMutex.Lock()
	defer driversMutex.Unlock()
	if _, exists := drivers[id]; exists {
		err = fmt.Errorf("duplicate registration for device: %v", id)
	} else {
		drivers[id] = v
	}
	return
}

// SetDriver gives a driver for a given list of devices (vendor, device pairs).
func SetDriver(v Driver, args ...interface{}) (err error) {
	var id DeviceID
	for _, a := range args {
		switch b := a.(type) {
		case VendorID:
			id.Vendor = b
		case VendorDeviceID:
			id.Device = b
			setDriver(v, id)
		case DeviceID:
			id = b
			setDriver(v, id)
		case []DeviceID:
			for i := range b {
				setDriver(v, b[i])
			}
		case []VendorDeviceID:
			for i := range b {
				setDriver(v, DeviceID{Vendor: id.Vendor, Device: b[i]})
			}
		}
	}
	return
}

func GetDriver(d DeviceID) Driver {
	driversMutex.Lock()
	defer driversMutex.Unlock()
	return drivers[d]
}

func (d *Device) ForeachCap(f func(h *CapabilityHeader, offset uint) (done bool, err error)) (err error) {
	r := d.GetDeviceConfig()
	o := uint(r.CapabilityOffset.Get(d))
	done := false
	for {
		var h CapabilityHeader
		h.Capability = Capability(d.ReadConfigUint8(o + 0))
		h.NextCapabilityHeader = U8(d.ReadConfigUint8(o + 1))
		done, err = f(&h, o)
		if err != nil || done {
			return
		}
		o = uint(h.NextCapabilityHeader)
		if o < 0x40 || o == 0xff {
			break
		}
	}
	return
}

func (d *Device) FindCap(c Capability) (offset uint, found bool) {
	d.ForeachCap(func(h *CapabilityHeader, o uint) (done bool, err error) {
		found = h.Capability == c
		if found {
			offset = o
			done = true
		}
		return
	})
	return
}

func (d *Device) GetCap(c Capability) (p unsafe.Pointer) {
	d.ForeachCap(func(h *CapabilityHeader, o uint) (done bool, err error) {
		if found := h.Capability == c; found {
			p = d.getRegs(o)
			done = true
		}
		return
	})
	return
}

func (d *Device) ForeachExtCap(f func(h *ExtCapabilityHeader, offset uint) (done bool, err error)) (err error) {
	o := uint(0x100)
	done := false
	for {
		var h ExtCapabilityHeader
		h.ExtCapability = ExtCapability(d.ReadConfigUint8(o+0)) | ExtCapability(d.ReadConfigUint8(o+1))<<8
		h.VersionAndNextOffset = U16(d.ReadConfigUint8(o+2)) | U16(d.ReadConfigUint8(o+3))<<8
		done, err = f(&h, o)
		if err != nil || done {
			return
		}
		o = uint(h.VersionAndNextOffset >> 4)
		if o < 0x100 || o == 0 {
			break
		}
	}
	return
}

func (d *Device) FindExtCap(c ExtCapability) (offset uint, found bool) {
	d.ForeachExtCap(func(h *ExtCapabilityHeader, o uint) (done bool, err error) {
		found = h.ExtCapability == c
		if found {
			offset = o
			done = true
		}
		return
	})
	return
}

func (d *Device) GetExtCap(c ExtCapability) (p unsafe.Pointer) {
	d.ForeachExtCap(func(h *ExtCapabilityHeader, o uint) (done bool, err error) {
		if found := h.ExtCapability == c; found {
			p = d.getRegs(o)
			done = true
		}
		return
	})
	return
}

//go:generate stringer -type=Capability,ExtCapability,HeaderType
