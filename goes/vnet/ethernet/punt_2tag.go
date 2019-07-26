// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ethernet

import (
	"github.com/platinasystems/go/vnet"

	"sync/atomic"
	"unsafe"
)

const (
	punt_2tag_next_punt uint = iota
	punt_2tag_next_error
)
const (
	punt_2tag_error_none uint = iota
	punt_2tag_error_not_double_tagged
	punt_2tag_error_unknown_disposition
)

type PuntConfig struct {
	// Next hop in vnet packet processing for this dispostion.
	Next string
	// Software interface and aux data to use for packet.
	RefOpaque vnet.RefOpaque
	// True to advance past vlan/2-vlan tags to layer 3 header.
	AdvanceL3Header     bool
	NReplaceVlanHeaders uint
	ReplaceVlanHeaders  [2]VlanTypeAndTag
}

type punt_packet_disposition struct {
	o            vnet.RefOpaque
	next         uint32
	data_advance int32
	header_index int32
	_            uint32 // pad
	replace_tags [8]byte
}

//go:generate gentemplate -d Package=ethernet -id punt_packet_disposition -d PoolType=punt_packet_disposition_pool -d Type=punt_packet_disposition -d Data=dispositions github.com/platinasystems/go/elib/pool.tmpl

type DoubleTaggedPuntNode vlan_tagged_punt_node

func (n *DoubleTaggedPuntNode) AddDisposition(cf PuntConfig) (i PuntDisposition) {
	return (*vlan_tagged_punt_node)(n).add_disposition(cf, 2)
}
func (n *DoubleTaggedPuntNode) DelDisposition(i PuntDisposition) (ok bool) {
	return (*vlan_tagged_punt_node)(n).del_disposition(i)
}
func PuntDispositionForTags(outer, inner VlanTag) (i PuntDisposition) {
	return PuntDisposition(inner.ToHost())<<16 | PuntDisposition(outer.ToHost())
}
func PuntTagsForDisposition(i PuntDisposition) (outer, inner VlanTag) {
	inner = VlanTag(i >> 16).FromHost()
	outer = VlanTag(i).FromHost()
	return
}

const sizeof_double_tag = 8

func (n *DoubleTaggedPuntNode) punt_x1(r0 *vnet.Ref) (next0 uint) {
	p0 := *(*vnet.Uint64)(r0.DataOffset(sizeof_header_no_type))
	q0 := (*[2]VlanTypeAndTag)(r0.DataOffset(sizeof_header_no_type))

	var t = (vnet.Uint64(TYPE_VLAN)<<48 | vnet.Uint64(TYPE_VLAN)<<16).FromHost()
	var m = (vnet.Uint64(0xffff)<<48 | vnet.Uint64(0xffff)<<16).FromHost()

	error0 := punt_2tag_error_none

	di0 := PuntDispositionForTags(q0[0].Tag, q0[1].Tag)

	if di0 >= PuntDisposition(n.punt_packet_disposition_pool.Len()) {
		error0 = punt_2tag_error_unknown_disposition
		di0 = 0
	}

	if p0&m != t {
		error0 = punt_2tag_error_not_double_tagged
	}

	d0 := &n.dispositions[di0]

	r0.RefOpaque = d0.o

	n.SetError(r0, error0)

	next0 = uint(d0.next)
	if error0 != punt_2tag_error_none {
		next0 = punt_2tag_next_error
	}

	// Possibly replace tag.
	*(*[2]VlanTypeAndTag)(r0.DataOffset(sizeof_header_no_type)) = *(*[2]VlanTypeAndTag)(unsafe.Pointer(&d0.replace_tags[0]))

	// Copy src and dst ethernet address.
	*(*header_no_type)(r0.DataOffset(uint(d0.header_index))) = *(*header_no_type)(r0.DataOffset(0))

	r0.Advance(int(d0.data_advance))

	return
}

func (n *DoubleTaggedPuntNode) punt_x2(r0, r1 *vnet.Ref) (next0, next1 uint) {
	p0 := *(*vnet.Uint64)(r0.DataOffset(sizeof_header_no_type))
	p1 := *(*vnet.Uint64)(r1.DataOffset(sizeof_header_no_type))
	q0 := (*[2]VlanTypeAndTag)(r0.DataOffset(sizeof_header_no_type))
	q1 := (*[2]VlanTypeAndTag)(r1.DataOffset(sizeof_header_no_type))

	var t = (vnet.Uint64(TYPE_VLAN)<<48 | vnet.Uint64(TYPE_VLAN)<<16).FromHost()
	var m = (vnet.Uint64(0xffff)<<48 | vnet.Uint64(0xffff)<<16).FromHost()

	error0, error1 := punt_2tag_error_none, punt_2tag_error_none

	di0 := PuntDispositionForTags(q0[0].Tag, q0[1].Tag)
	di1 := PuntDispositionForTags(q1[0].Tag, q1[1].Tag)

	if di0 >= PuntDisposition(n.punt_packet_disposition_pool.Len()) {
		error0 = punt_2tag_error_unknown_disposition
		di0 = 0
	}
	if di1 >= PuntDisposition(n.punt_packet_disposition_pool.Len()) {
		error1 = punt_2tag_error_unknown_disposition
		di1 = 0
	}

	if p0&m != t {
		error0 = punt_2tag_error_not_double_tagged
	}
	if p1&m != t {
		error1 = punt_2tag_error_not_double_tagged
	}

	d0, d1 := &n.dispositions[di0], &n.dispositions[di1]

	r0.RefOpaque = d0.o
	r1.RefOpaque = d1.o

	n.SetError(r0, error0)
	n.SetError(r1, error1)

	next0, next1 = uint(d0.next), uint(d1.next)
	if error0 != punt_2tag_error_none {
		next0 = punt_2tag_next_error
	}
	if error1 != punt_2tag_error_none {
		next1 = punt_2tag_next_error
	}

	// Possibly replace tag.
	*(*[2]VlanTypeAndTag)(r0.DataOffset(sizeof_header_no_type)) = *(*[2]VlanTypeAndTag)(unsafe.Pointer(&d0.replace_tags[0]))
	*(*[2]VlanTypeAndTag)(r1.DataOffset(sizeof_header_no_type)) = *(*[2]VlanTypeAndTag)(unsafe.Pointer(&d1.replace_tags[0]))

	// Copy src and dst ethernet address.
	*(*header_no_type)(r0.DataOffset(uint(d0.header_index))) = *(*header_no_type)(r0.DataOffset(0))
	*(*header_no_type)(r1.DataOffset(uint(d1.header_index))) = *(*header_no_type)(r1.DataOffset(0))

	r0.Advance(int(d0.data_advance))
	r1.Advance(int(d1.data_advance))

	return
}

func (n *DoubleTaggedPuntNode) Init(v *vnet.Vnet, name string) {
	n.Next = []string{
		punt_2tag_next_error: "error",
		punt_2tag_next_punt:  "punt",
	}
	n.Errors = []string{
		punt_2tag_error_none:                "no error",
		punt_2tag_error_not_double_tagged:   "not double vlan tagged",
		punt_2tag_error_unknown_disposition: "unknown packet disposition",
	}
	v.RegisterInOutNode(n, name+"-double-tagged-punt")
	d := n.AddDisposition(PuntConfig{Next: "punt"})
	if d != 0 {
		panic("must be zero")
	}
}

func (n *DoubleTaggedPuntNode) NodeInput(in *vnet.RefIn, o *vnet.RefOut) {
	q := n.GetEnqueue(in)
	i, n_left := in.Range()

	for n_left >= 2 {
		r0, r1 := in.Get2(i)
		x0, x1 := n.punt_x2(r0, r1)
		q.Put2(r0, r1, x0, x1)
		n_left -= 2
		i += 2
	}

	for n_left >= 1 {
		r0 := in.Get1(i)
		x0 := n.punt_x1(r0)
		q.Put1(r0, x0)
		n_left -= 1
		i += 1
	}
}

type inject_next uint32

func (n inject_next) get(next_offset uint) (next uint) {
	next = uint((n >> 1) + (inject_next(next_offset) & -(n & 1)))
	return
}
func (n *inject_next) set(next uint, offset_valid bool) {
	v := inject_next(2 * next)
	if offset_valid {
		v += 1
	}
	*n = v
}

type inject_packet_disposition struct {
	next inject_next
	tags [2]VlanTag
}

//go:generate gentemplate -d Package=ethernet -id inject_packet_disposition -d VecType=inject_packet_disposition_vec -d Type=inject_packet_disposition github.com/platinasystems/go/elib/vec.tmpl

type DoubleTaggedInjectNode struct {
	vnet.InOutNode
	disposition_by_si inject_packet_disposition_vec
	sequence          uint32
}

func (n *DoubleTaggedInjectNode) AddDisposition(next uint, offset_valid bool, si vnet.Si, tags [2]VlanTag) {
	n.sw_if_add_del(n.Vnet, si, true)
	d := &n.disposition_by_si[si]
	d.tags = tags
	d.next.set(next, offset_valid)
	return
}

func (n *DoubleTaggedInjectNode) sw_if_add_del(v *vnet.Vnet, si vnet.Si, isUp bool) (err error) {
	var zero inject_packet_disposition
	zero.next.set(uint(inject_2tag_next_error), false)
	n.disposition_by_si.ValidateInit(uint(si), zero)
	return
}

func (n *DoubleTaggedInjectNode) inject_x1(r0 *vnet.Ref, next_offset uint) (next0 uint) {
	var t = (vnet.Uint64(TYPE_VLAN)<<48 | vnet.Uint64(TYPE_VLAN)<<16).FromHost()

	d0 := &n.disposition_by_si[r0.Si]

	o0 := t
	if vnet.HostIsNetworkByteOrder() {
		o0 |= vnet.Uint64(d0.tags[0]) | vnet.Uint64(d0.tags[1])<<32
	} else {
		o0 |= vnet.Uint64(d0.tags[0])<<16 | vnet.Uint64(d0.tags[1])<<48
	}

	h0 := *(*header_no_type)(r0.DataOffset(0))

	r0.Advance(-sizeof_double_tag)

	*(*vnet.Uint64)(r0.DataOffset(sizeof_header_no_type)) = o0

	*(*header_no_type)(r0.DataOffset(0)) = h0

	n.SetError(r0, inject_2tag_error_unknown_interface)

	next0 = d0.next.get(next_offset)

	return
}
func (n *DoubleTaggedInjectNode) inject_x2(r0, r1 *vnet.Ref, next_offset uint) (next0, next1 uint) {
	var t = (vnet.Uint64(TYPE_VLAN)<<48 | vnet.Uint64(TYPE_VLAN)<<16).FromHost()

	d0, d1 := &n.disposition_by_si[r0.Si], &n.disposition_by_si[r1.Si]

	o0, o1 := t, t
	if vnet.HostIsNetworkByteOrder() {
		o0 |= vnet.Uint64(d0.tags[0]) | vnet.Uint64(d0.tags[1])<<32
		o1 |= vnet.Uint64(d1.tags[0]) | vnet.Uint64(d1.tags[1])<<32
	} else {
		o0 |= vnet.Uint64(d0.tags[0])<<16 | vnet.Uint64(d0.tags[1])<<48
		o1 |= vnet.Uint64(d1.tags[0])<<16 | vnet.Uint64(d1.tags[1])<<48
	}

	h0, h1 := *(*header_no_type)(r0.DataOffset(0)), *(*header_no_type)(r1.DataOffset(0))

	r0.Advance(-sizeof_double_tag)
	r1.Advance(-sizeof_double_tag)

	*(*vnet.Uint64)(r0.DataOffset(sizeof_header_no_type)) = o0
	*(*vnet.Uint64)(r1.DataOffset(sizeof_header_no_type)) = o1

	*(*header_no_type)(r0.DataOffset(0)) = h0
	*(*header_no_type)(r1.DataOffset(0)) = h1

	n.SetError(r0, inject_2tag_error_unknown_interface)
	n.SetError(r1, inject_2tag_error_unknown_interface)

	next0 = d0.next.get(next_offset)
	next1 = d1.next.get(next_offset)

	return
}

const (
	inject_2tag_next_error uint = iota
)
const (
	inject_2tag_error_none uint = iota
	inject_2tag_error_unknown_interface
)

func (n *DoubleTaggedInjectNode) Init(v *vnet.Vnet, name string) {
	n.Next = []string{
		inject_2tag_next_error: "error",
	}
	n.Errors = []string{
		inject_2tag_error_none:              "no error",
		inject_2tag_error_unknown_interface: "unknown interface",
	}
	v.RegisterInOutNode(n, name+"-double-tagged-inject")
	v.RegisterSwIfAddDelHook(n.sw_if_add_del)
}

func (n *DoubleTaggedInjectNode) NodeInput(in *vnet.RefIn, o *vnet.RefOut) {
	q := n.GetEnqueue(in)
	i, n_left := in.Range()
	next_offset := uint(atomic.AddUint32(&n.sequence, 1)) & 1

	for n_left >= 2 {
		r0, r1 := in.Get2(i)
		x0, x1 := n.inject_x2(r0, r1, next_offset)
		q.Put2(r0, r1, x0, x1)
		n_left -= 2
		i += 2
	}

	for n_left >= 1 {
		r0 := in.Get1(i)
		x0 := n.inject_x1(r0, next_offset)
		q.Put1(r0, x0)
		n_left -= 1
		i += 1
	}
}

type DoubleTaggedPuntInjectNodes struct {
	Punt   DoubleTaggedPuntNode
	Inject DoubleTaggedInjectNode
}

func (n *DoubleTaggedPuntInjectNodes) Init(v *vnet.Vnet, name string) {
	n.Punt.Init(v, name)
	n.Inject.Init(v, name)
}
