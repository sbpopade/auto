// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package mod

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/platinasystems/go/goes/cmd/ip/internal/group"
	"github.com/platinasystems/go/goes/cmd/ip/internal/options"
	"github.com/platinasystems/go/goes/lang"
	"github.com/platinasystems/go/internal/netns"
	"github.com/platinasystems/go/internal/nl"
	"github.com/platinasystems/go/internal/nl/rtnl"
)

const nogroup = ^uint32(0)

type Command string

type mod struct {
	name string
	args []string
	opt  *options.Options

	sr *nl.SockReceiver

	hdr   nl.Hdr
	msg   rtnl.IfInfoMsg
	attrs nl.Attrs

	tinfo nl.Attrs

	ifindexByName    map[string]int32
	ifindicesByGroup map[uint32][]int32

	indices []int32

	netns *os.File
}

func (c Command) String() string { return string(c) }

func (c Command) Usage() string {
	return fmt.Sprint("ip link ", c, ` SUBJECT [ OPTION... ]`)
}

func (Command) Apropos() lang.Alt {
	return lang.Alt{
		lang.EnUS: "link attributes",
	}
}

func (Command) Man() lang.Alt {
	return lang.Alt{
		lang.EnUS: Man,
	}
}

func (c Command) Main(args ...string) error {
	m := mod{name: string(c)}

	sock, err := nl.NewSock()
	if err != nil {
		return err
	}
	defer sock.Close()

	m.sr = nl.NewSockReceiver(sock)

	if err = m.getifindices(); err != nil {
		return err
	}

	m.opt, m.args = options.New(args)
	if err = m.parse(); err != nil {
		return err
	}
	if m.netns != nil {
		defer m.netns.Close()
	}

	m.hdr.Flags = nl.NLM_F_REQUEST | nl.NLM_F_ACK
	m.msg.Family = rtnl.AF_UNSPEC
	switch c {
	case "change", "set":
		m.hdr.Type = rtnl.RTM_NEWLINK
	case "replace":
		m.hdr.Type = rtnl.RTM_NEWLINK
		m.hdr.Flags |= nl.NLM_F_CREATE | nl.NLM_F_REPLACE
	default:
		return fmt.Errorf("%s: unknown", c)
	}

	for _, ifindex := range m.indices {
		m.msg.Index = ifindex
		if req, err := nl.NewMessage(
			m.hdr,
			m.msg,
			m.attrs...,
		); err != nil {
			return err
		} else if err = m.sr.UntilDone(req, nl.DoNothing); err != nil {
			return err
		}
	}
	return nil
}

func (m *mod) getifindices() error {
	m.ifindexByName = make(map[string]int32)
	m.ifindicesByGroup = make(map[uint32][]int32)

	req, err := nl.NewMessage(
		nl.Hdr{
			Type:  rtnl.RTM_GETLINK,
			Flags: nl.NLM_F_REQUEST | nl.NLM_F_DUMP,
		},
		rtnl.IfInfoMsg{
			Family: rtnl.AF_UNSPEC,
		},
	)
	if err != nil {
		return err
	}
	return m.sr.UntilDone(req, func(b []byte) {
		var ifla rtnl.Ifla
		if nl.HdrPtr(b).Type != rtnl.RTM_NEWLINK {
			return
		}
		msg := rtnl.IfInfoMsgPtr(b)
		ifla.Write(b)
		name := nl.Kstring(ifla[rtnl.IFLA_IFNAME])
		m.ifindexByName[name] = msg.Index
		gid := nl.Uint32(ifla[rtnl.IFLA_GROUP])
		m.ifindicesByGroup[gid] = append(m.ifindicesByGroup[gid],
			msg.Index)
	})
}

func (m *mod) parse() error {
	var err error
	var dev, link int32
	var gid uint32

	m.args = m.opt.Flags.More(m.args,
		[]string{"up", "+up"},
		[]string{"down", "no-up", "-up"},
		[]string{"no-master", "-master"},
		[]string{"arp", "+arp"},
		[]string{"no-arp", "-arp"},
		[]string{"dynamic", "+dynamic"},
		[]string{"no-dynamic", "-dynamic"},
		[]string{"multicast", "+multicast"},
		[]string{"no-multicast", "-multicast"},
		[]string{"allmulticast", "+allmulticast"},
		[]string{"no-allmulticast", "-allmulticast"},
		[]string{"promisc", "+promisc"},
		[]string{"no-promisc", "-promisc"},
		[]string{"trailers", "+trailers"},
		[]string{"no-trailers", "-trailers"},
		[]string{"carrier", "+carrier"},
		[]string{"no-carrier", "-carrier"},
		[]string{"protodown", "+protodown"},
		[]string{"no-protodown", "-protodown"},
		[]string{"no-master", "-master"},
		[]string{"no-vrf", "-vrf"},
	)
	m.args = m.opt.Parms.More(m.args,
		"dev",
		"link",
		"index",
		"group",
		"addrgenmode",
		"vf",
		"name",
		"alias",
		"qdisc",
		"mtu",
		"address",
		"master",
		"vrf",
		"link-netnsid",
		"netns",
		"mode",
		"state",
		[]string{"broadcast", "brd"},
		"numrxqueues",
		"numtxqueues",
		[]string{"txqueuelen", "qlen", "txqlen"},
	)

	for _, x := range []struct {
		set   string
		unset string
		flag  uint32
	}{
		{"up", "down", rtnl.IFF_UP},
		{"allmulticast", "no-allmulticast", rtnl.IFF_ALLMULTI},
		{"no-arp", "arp", rtnl.IFF_NOARP},
		{"dynamic", "no-dynamic", rtnl.IFF_DYNAMIC},
		{"multicast", "no-multicast", rtnl.IFF_MULTICAST},
		{"promisc", "no-promisc", rtnl.IFF_PROMISC},
		{"no-trailers", "trailers", rtnl.IFF_NOTRAILERS},
	} {
		if m.opt.Flags.ByName[x.set] {
			m.msg.Change |= x.flag
			m.msg.Flags |= x.flag
		} else if m.opt.Flags.ByName[x.unset] {
			m.msg.Change |= x.flag
			m.msg.Flags &^= x.flag
		}
	}
	for _, x := range []struct {
		set   string
		unset string
		t     uint16
	}{
		{"carrier", "no-carrier", rtnl.IFLA_CARRIER},
		{"protodown", "no-protodown", rtnl.IFLA_PROTO_DOWN},
	} {
		if m.opt.Flags.ByName[x.set] {
			m.attrs = append(m.attrs, nl.Attr{x.t, nl.Uint8Attr(1)})
		} else if m.opt.Flags.ByName[x.unset] {
			m.attrs = append(m.attrs, nl.Attr{x.t, nl.Uint8Attr(0)})
		}
	}
	for _, x := range []struct {
		name string
		t    uint16
	}{
		{"no-master", rtnl.IFLA_MASTER},
		{"no-vrf", rtnl.IFLA_MASTER},
	} {
		if m.opt.Flags.ByName[x.name] {
			m.attrs = append(m.attrs, nl.Attr{x.t, nl.Int32Attr(0)})
		}
	}
	for _, x := range []struct {
		name string
		p    *int32
	}{
		{"dev", &dev},
		{"link", &link},
	} {
		if s := m.opt.Parms.ByName[x.name]; len(s) > 0 {
			var found bool
			*x.p, found = m.ifindexByName[s]
			if !found {
				return fmt.Errorf("%s: %q not found",
					x.name, s)
			}
		}
	}
	if s := m.opt.Parms.ByName["index"]; m.name == "add" && len(s) > 0 {
		var ifindex int32
		if _, err = fmt.Sscan(s, &ifindex); err != nil {
			return fmt.Errorf("index: %q %v", s, err)
		}
		m.indices = []int32{ifindex}
	}
	if s := m.opt.Parms.ByName["group"]; len(s) > 0 {
		if gid = group.Id(s); gid == nogroup {
			return fmt.Errorf("group: %q not found", s)
		}
	} else {
		gid = nogroup
	}
	for _, x := range []struct {
		name  string
		parse func(string) error
	}{
		{"addrgenmode", m.parseAddrGenMode},
		{"vf", m.parseVf},
	} {
		if s := m.opt.Parms.ByName[x.name]; len(s) > 0 {
			if err = x.parse(s); err != nil {
				return err
			}
		}
	}
	for _, x := range []struct {
		name string
		t    uint16
	}{
		{"name", rtnl.IFLA_IFNAME},
		{"alias", rtnl.IFLA_IFALIAS},
		{"qdisc", rtnl.IFLA_QDISC},
	} {
		s := m.opt.Parms.ByName[x.name]
		if len(s) == 0 {
			continue
		}
		m.attrs = append(m.attrs, nl.Attr{x.t, nl.KstringAttr(s)})
	}
	for _, x := range []struct {
		name string
		t    uint16
	}{
		{"link-netnsid", rtnl.IFLA_LINK_NETNSID},
		{"mtu", rtnl.IFLA_MTU},
		{"numtxqueues", rtnl.IFLA_NUM_TX_QUEUES},
		{"numrxqueues", rtnl.IFLA_NUM_RX_QUEUES},
		{"txqueuelen", rtnl.IFLA_TXQLEN},
	} {
		var u32 uint32
		s := m.opt.Parms.ByName[x.name]
		if len(s) == 0 {
			continue
		}
		_, err := fmt.Sscan(s, &u32)
		if err != nil {
			return fmt.Errorf("%s: %q %v", x.name, s, err)
		}
		m.attrs = append(m.attrs, nl.Attr{x.t, nl.Uint32Attr(u32)})
	}
	for _, x := range []struct {
		name string
		t    uint16
	}{
		{"address", rtnl.IFLA_ADDRESS},
		{"broadcast", rtnl.IFLA_BROADCAST},
	} {
		s := m.opt.Parms.ByName[x.name]
		if len(s) == 0 {
			continue
		}
		mac, err := net.ParseMAC(s)
		if err != nil {
			return fmt.Errorf("%s: %q %v", x.name, s, err)
		}
		m.attrs = append(m.attrs, nl.Attr{x.t, nl.BytesAttr(mac)})
	}
	for _, x := range []struct {
		name string
		t    uint16
	}{
		{"master", rtnl.IFLA_MASTER},
		{"vrf", rtnl.IFLA_MASTER},
	} {
		s := m.opt.Parms.ByName[x.name]
		if len(s) == 0 {
			continue
		}
		ifindex, found := m.ifindexByName[s]
		if !found {
			return fmt.Errorf("%s: %q not found", x.name, s)
		}
		m.attrs = append(m.attrs, nl.Attr{x.t, nl.Int32Attr(ifindex)})
	}
	if s := m.opt.Parms.ByName["netns"]; len(s) > 0 {
		var id int32
		var t uint16
		m.netns, err = os.Open(filepath.Join("/var/run/netns", s))
		if err == nil {
			t = rtnl.IFLA_NET_NS_FD
			id = int32(m.netns.Fd())
		} else if _, err := fmt.Sscan(s, &id); err != nil {
			return fmt.Errorf("netns: %q %v", s, err)
		} else {
			t = rtnl.IFLA_NET_NS_PID
		}
		m.attrs = append(m.attrs, nl.Attr{t, nl.Int32Attr(id)})
	}
	if s := m.opt.Parms.ByName["mode"]; len(s) > 0 {
		mode, found := rtnl.IfLinkModeByName[s]
		if !found {
			return fmt.Errorf("mode: %q unknown", s)
		}
		m.attrs = append(m.attrs, nl.Attr{rtnl.IFLA_LINKMODE,
			nl.Uint8Attr(mode)})
	}
	if s := m.opt.Parms.ByName["state"]; len(s) > 0 {
		u8, found := rtnl.IfOperByName[s]
		if !found {
			return fmt.Errorf("state: %q unknown", s)
		}
		m.attrs = append(m.attrs, nl.Attr{rtnl.IFLA_OPERSTATE,
			nl.Uint8Attr(u8)})
	}
	if m.name == "add" {
		switch len(m.args) {
		case 0:
		case 1:
			m.attrs = append(m.attrs, nl.Attr{rtnl.IFLA_IFNAME,
				nl.KstringAttr(m.args[0])})
		default:
			return fmt.Errorf("%v: unexpected", m.args[1:])
		}
		if link != 0 {
			m.attrs = append(m.attrs, nl.Attr{rtnl.IFLA_LINK,
				nl.Int32Attr(link)})
		}
		if len(m.indices) == 0 {
			m.indices = []int32{0}
		}
		return nil
	}
	switch len(m.args) {
	case 0:
		if dev != 0 {
			m.indices = []int32{dev}
		} else if gid != nogroup {
			m.indices = m.ifindicesByGroup[gid]
			if len(m.indices) == 0 {
				return fmt.Errorf("group %d empty", gid)
			}
		} else {
			return fmt.Errorf("no dev | group")
		}
	case 1:
		if ifindex, found := m.ifindexByName[m.args[0]]; !found {
			return fmt.Errorf("dev: %q not found", m.args[0])
		} else {
			m.indices = []int32{ifindex}
		}
		if gid != nogroup {
			m.attrs = append(m.attrs, nl.Attr{rtnl.IFLA_GROUP,
				nl.Uint32Attr(gid)})
		}
	default:
		return fmt.Errorf("%v: unexpected", m.args[1:])
	}
	return nil
}

func (m *mod) parseAddrGenMode(s string) error {
	mode, found := rtnl.In6AddrGenModeByName[s]
	if !found {
		return fmt.Errorf("addrgenmode: %q unknown", s)
	}
	m.attrs = append(m.attrs, nl.Attr{rtnl.IFLA_AF_SPEC,
		nl.Attr{uint16(rtnl.AF_INET6),
			nl.Attr{rtnl.IFLA_INET6_ADDR_GEN_MODE,
				nl.Uint8Attr(mode),
			},
		},
	},
	)
	return nil
}

func (Command) Complete(args ...string) (list []string) {
	var larg, llarg string
	n := len(args)
	if n > 0 {
		larg = args[n-1]
	}
	if n > 1 {
		llarg = args[n-2]
	}
	cpv := options.CompleteParmValue
	cpv["dev"] = options.CompleteIfName
	cpv["link"] = options.CompleteIfName
	cpv["group"] = group.Complete
	cpv["name"] = options.NoComplete
	cpv["alias"] = options.NoComplete
	cpv["txqlen"] = options.NoComplete
	cpv["txqueuelen"] = options.NoComplete
	cpv["mtu"] = options.NoComplete
	cpv["address"] = options.NoComplete
	cpv["broadcast"] = options.NoComplete
	cpv["brd"] = options.NoComplete
	cpv["peer"] = options.NoComplete
	cpv["master"] = options.CompleteIfName
	cpv["addrgenmode"] = rtnl.CompleteIn6AddrGenMode
	cpv["netns"] = netns.CompleteName
	cpv["link-netnsid"] = options.NoComplete
	cpv["vf"] = options.NoComplete
	cpv["xdp"] = options.NoComplete
	if method, found := cpv[llarg]; found {
		list = method(larg)
	} else {
		for _, name := range append(options.CompleteOptNames,
			"dev",
			"link",
			"group",
			"name",
			"alias",
			"txqlen",
			"txqueuelen",
			"mtu",
			"address",
			"brd",
			"broadcast",
			"peer",
			"master",
			"addrgenmode",
			"netns",
			"link-netnsid",
			"vf",
			"xdp",
		) {
			if len(larg) == 0 || strings.HasPrefix(name, larg) {
				list = append(list, name)
			}
		}
	}
	return
}
