// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unix

import (
	"github.com/platinasystems/go/elib"
	"github.com/platinasystems/go/elib/elog"
	"github.com/platinasystems/go/elib/iomux"
	"github.com/platinasystems/go/vnet"
	"github.com/platinasystems/go/vnet/ethernet"

	"sync/atomic"
	"syscall"
	"fmt"
)

type tx_node struct {
	vnet.OutputNode
	m       *net_namespace_main
	pv_pool chan *tx_packet_vector
	p_pool  chan *tx_packet
}

type tuntap_interface_tx_node struct {
	active_refs  int32
	to_tx        chan *tx_packet_vector
	to_tx_thread chan struct{}
	pv           *tx_packet_vector
}

func (i *tuntap_interface_tx_node) tx_stop() {
	if i.to_tx != nil {
		close(i.to_tx)
		i.to_tx = nil
	}
	if i.to_tx_thread != nil {
		close(i.to_tx_thread)
		i.to_tx_thread = nil
	}
}

func (i *tuntap_interface) tx_start() {
	i.to_tx = make(chan *tx_packet_vector, vnet.MaxOutstandingTxRefs)
	i.to_tx_thread = make(chan struct{}, 64)
	go i.tx_thread()
}

type tx_packet struct {
	ifindex uint32
	iovs    iovecVec
}

func (p *tx_packet) add_one_ref(r *vnet.Ref, iʹ, lʹ uint) (i, l uint) {
	i, l = iʹ, lʹ
	d := r.DataLen()
	p.iovs.Validate(i)
	p.iovs[i].Base = (*byte)(r.Data())
	p.iovs[i].Len = uint64(d)
	return i + 1, l + d
}

func (p *tx_packet) add_ref(r *vnet.Ref, ifindex uint32) (i, l uint) {
	p.ifindex = ifindex
	for {
		i, l = p.add_one_ref(r, i, l)
		if r.NextValidFlag() == 0 {
			break
		}
		r = r.NextRef()
	}
	p.iovs = p.iovs[:i]
	return
}

type tx_packet_vector struct {
	n_packets   uint
	n_refs      uint
	intf        *tuntap_interface
	ri          *vnet.RefIn
	buffer_pool *vnet.BufferPool
	a           [tx_packet_vector_max_len]syscall.RawSockaddrLinklayer
	m           [tx_packet_vector_max_len]mmsghdr
	p           [tx_packet_vector_max_len]tx_packet
	r           [tx_packet_vector_max_len]vnet.Ref
}

func (n *tx_node) get_packet_vector(p *vnet.BufferPool, intf *tuntap_interface) (v *tx_packet_vector) {
	select {
	case v = <-n.pv_pool:
	default:
		v = &tx_packet_vector{}
	}
	v.n_packets = 0
	v.n_refs = 0
	v.buffer_pool = p
	v.intf = intf
	return
}
func (n *tx_node) put_packet_vector(v *tx_packet_vector) { n.pv_pool <- v }

func (v *tx_packet_vector) add_packet(n *tx_node, r *vnet.Ref, intf *tuntap_interface) {
	ifindex := intf.ifindex
	i := v.n_packets
	v.n_packets++

	// For tun interfaces strip ethernet header.
	// First 4 bits of ip header will indicate ip4/ip6 packet type.
	if intf.isTun {
		r.Advance(ethernet.SizeofHeader)
	}

	p := &v.p[i]
	nr, l := p.add_ref(r, ifindex)
	v.r[i] = *r
	v.n_refs += nr

	a := &v.a[i]
	*a = raw_sockaddr_ll_template
	a.Ifindex = int32(p.ifindex)
	if i > 0 {
		v.m[i-1].msg_hdr.Flags |= syscall.MSG_MORE
	}
	v.m[i].msg_hdr.set(a, p.iovs, 0)
	v.m[i].msg_len = uint32(l)
}

const (
	tx_error_none = iota
	tx_error_unknown_interface
	tx_error_interface_down
	tx_error_packet_too_large
	tx_error_drop
)

func (n *tx_node) init(m *net_namespace_main) {
	n.m = m
	n.Errors = []string{
		tx_error_unknown_interface: "unknown interface",
		tx_error_interface_down:    "interface is down",
		tx_error_packet_too_large:  "packet too large",
		tx_error_drop:              "error drops",
	}
	m.m.v.RegisterOutputNode(n, "punt")
	n.pv_pool = make(chan *tx_packet_vector, 4*vnet.MaxVectorLen)
}

func (n *tx_node) NodeOutput(out *vnet.RefIn) {
	elog.F1u("unix tx output %d packets", uint64(out.InLen()))
	var (
		pv      *tx_packet_vector
		pv_intf *tuntap_interface
	)
	for i := uint(0); i < out.InLen(); i++ {
		ref := &out.Refs[i]
		intf, ok := n.m.vnet_tuntap_interface_by_si[ref.Si]
		if !ok {
			out.BufferPool.FreeRefs(ref, 1, true)
			n.CountError(tx_error_unknown_interface, 1)
			continue
		}

		//Strip away outer vlan header if packet is a vlan with vlan id of 0
		{
			// Get the next 4 bytes after 12 bytes of src/dst ethernet address
			p0 := (*ethernet.VlanTypeAndTag)(ref.DataOffset(12))
			if false { //debug print
				p1 := (*ethernet.VlanTypeAndTag)(ref.DataOffset(16))  // not really VLAN, but just want to get next 4 byte after VLAN
				s := ""
				if p0.Type.IsVlan() {
					s = fmt.Sprintf(", tag=%v, inner type=%v", p0.Tag.Id(), p1.Type.FromHost())
				}
				fmt.Printf("punt: type=%v%s\n", p0.Type.FromHost(), s)
				fmt.Printf("  ref.Si=%v intf=%v, ok=%t \n", ref.Si, intf, ok)
			}
			if p0.Type.IsVlan() && p0.Tag.Id() == 0 {
				// Copy src/dst ethernet address
				h0 := *(*ethernet.HeaderNoType)(ref.DataOffset(0))
				// Move it 4 bytes forward (overwriting the 4 bytes of vlan)
				*(*ethernet.HeaderNoType)(ref.DataOffset(4)) = h0
				// Advance 4 bytes
				ref.Advance(4)
				if false { //debug print
					p0 := (*ethernet.VlanTypeAndTag)(ref.DataOffset(12))
					fmt.Printf("  stripped vlan 0: type=%v\n", p0.Type.FromHost())
				}
			}
		}

		if intf != pv_intf {
			if pv != nil {
				pv.tx_queue(n, out)
				pv = nil
			}
			pv_intf = intf
		}
		if pv == nil {
			pv = n.get_packet_vector(out.BufferPool, intf)
		}
		pv.add_packet(n, ref, intf)
		if pv.n_packets >= uint(len(pv.p)) {
			pv.tx_queue(n, out)
			pv = nil
			pv_intf = nil
		}
	}
	if pv != nil {
		pv.tx_queue(n, out)
	}
}

func (v *tx_packet_vector) tx_queue(n *tx_node, ri *vnet.RefIn) {
	np, intf := v.n_packets, v.intf

	if intf.Fd == -1 {
		v.buffer_pool.FreeRefs(&v.r[0], np, true)
		n.put_packet_vector(v)
		n.CountError(tx_error_interface_down, np)
		return
	}

	v.ri = ri
	n.AddSuspendActivity(ri, int(v.n_refs))
	if da := int32(v.n_refs); da == atomic.AddInt32(&intf.active_refs, da) {
		iomux.Update(intf)
	}
	select {
	case intf.to_tx <- v:
		if elog.Enabled() {
			e := rx_tx_elog{
				kind:      tx_elog_queue_write,
				name:      intf.elog_name,
				n_packets: uint32(np),
				n_refs:    uint32(v.n_refs),
			}
			elog.Add(&e)
		}
		return
	default:
		// Should never happen since to_tx has MaxOutstandingTxRefs entries.
		panic("tx full")
	}
}

func (pv *tx_packet_vector) advance(n *tx_node, intf *tuntap_interface, i uint) (done bool) {
	np := pv.n_packets
	n_left := np - i
	pv.buffer_pool.FreeRefs(&pv.r[0], i, true)

	// Did we send all packets in vector?
	if done = n_left == 0; done {
		n.put_packet_vector(pv)
		intf.pv = nil
		return
	}
	// Otherwise copy unsent packets back to vector; we'll return to it later.
	copy(pv.a[:n_left], pv.a[i:])
	copy(pv.m[:n_left], pv.m[i:])
	copy(pv.r[:n_left], pv.r[i:])

	// For packets swap them to avoid leaking iovecs.
	for j := uint(0); j < n_left; j++ {
		pv.p[j], pv.p[i+j] = pv.p[i+j], pv.p[j]
	}
	pv.n_packets = n_left
	return
}

func (intf *tuntap_interface) write_packet_vector() (n_packets, n_refs, n_drops uint, would_block bool) {
	ns := intf.namespace
	n := &ns.m.tx_node
	pv := intf.pv
loop:
	for i := uint(0); i < pv.n_packets; i++ {
		var errno syscall.Errno

		// Try to send packet either with sendmsg if present or fall back to writev.
		for {
			// First try sendmsg.
			if !ns.m.tuntap_sendmsg_recvmsg_disable {
				// sendmsg/sendmmsg does yet not work on /dev/net/tun sockets.  ENOTSOCK
				_, errno = sendmsg(intf.Fd, 0, &pv.m[i].msg_hdr)
				ns.m.tuntap_sendmsg_recvmsg_disable = errno == syscall.ENOTSOCK
				if !ns.m.tuntap_sendmsg_recvmsg_disable {
					break
				}
			} else {
				// Use writev since sendmsg failed.
				_, errno = writev(intf.Fd, pv.p[i].iovs)
				break
			}
		}

		// Keep current packet when blocking.  We'll send it later.
		if would_block = errno == syscall.EWOULDBLOCK; would_block {
			break loop
		}

		n_refs += pv.p[i].iovs.Len()
		switch errno {
		case syscall.EIO:
			// Signaled by tun.c in kernel and means that interface is down.
			n.CountError(tx_error_interface_down, 1)
			n_drops++
		case syscall.EMSGSIZE:
			n.CountError(tx_error_packet_too_large, 1)
			n_drops++
		default:
			if errno != 0 {
				n.CountError(tx_error_drop, 1)
				n_drops++
			}
			n_packets++
		}
	}

	if elog.Enabled() {
		e := rx_tx_elog{
			kind:        tx_elog_write,
			name:        intf.elog_name,
			n_packets:   uint32(n_packets),
			n_refs:      uint32(n_refs),
			n_drops:     uint32(n_drops),
			would_block: would_block,
		}
		elog.Add(&e)
	}

	n.AddSuspendActivity(pv.ri, -int(n_refs))

	// Advance to next packet in error case.
	if n_advance := n_packets + n_drops; n_advance > 0 {
		pv.advance(n, intf, n_advance)
	}
	return
}

func (intf *tuntap_interface) WriteAvailable() bool { return intf.active_refs > 0 }
func (intf *tuntap_interface) WriteReady() (err error) {
	select {
	case intf.to_tx_thread <- struct{}{}:
	default:
	}
	return
}

func (intf *tuntap_interface) tx_thread() {
	for range intf.to_tx_thread {
		intf.tx_write()
	}
}

func (intf *tuntap_interface) tx_write() {
	n_refs, n_packets, n_drops := uint(0), uint(0), uint(0)

loop:
	for {
		if intf.pv == nil {
			select {
			case intf.pv = <-intf.to_tx:
			default:
				break loop
			}
		}
		np, nr, nd, would_block := intf.write_packet_vector()
		n_packets += np
		n_refs += nr
		n_drops += nd
		if 0 == atomic.AddInt32(&intf.active_refs, -int32(nr)) {
			iomux.Update(intf)
		}
		if would_block {
			break loop
		}
	}
	if n_packets == 0 {
		return
	}

	// Count punts and drops on this interface.
	{
		n := &intf.namespace.m.tx_node
		th := n.Vnet.GetIfThread(0)
		vnet.IfPunts.Add(th, intf.si, n_packets)
		vnet.IfDrops.Add(th, intf.si, n_drops)
	}
	return
}

const (
	tx_elog_queue_write = iota
	tx_elog_write
	rx_elog_input
	rx_elog_read
)

type rx_tx_elog_kind uint32

func (k rx_tx_elog_kind) String() string {
	t := [...]string{
		tx_elog_queue_write: "tx queue-write",
		tx_elog_write:       "tx write",
		rx_elog_input:       "rx input",
		rx_elog_read:        "rx read",
	}
	return elib.StringerHex(t[:], int(k))
}

type rx_tx_elog struct {
	kind        rx_tx_elog_kind
	name        elog.StringRef
	n_packets   uint32
	n_refs      uint32
	n_drops     uint32
	would_block bool
}

func (e *rx_tx_elog) Elog(l *elog.Log) {
	blocked := ""
	if e.would_block {
		blocked = " would-block"
	}
	switch e.kind {
	case tx_elog_queue_write, tx_elog_write, rx_elog_read:
		if e.n_drops != 0 {
			l.Logf("unix %s %s%s %d packets, %d refs, %d drops", e.kind, e.name, blocked,
				e.n_packets, e.n_refs, e.n_drops)
		} else {
			l.Logf("unix %s %s%s %d packets, %d refs", e.kind, e.name, blocked,
				e.n_packets, e.n_refs)
		}
	case rx_elog_input: // no interface name for input
		l.Logf("unix %s %d packets", e.kind, e.n_packets)
	}
}
