// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package rtnl

import (
	"sort"
	"strings"
	"syscall"
)

var ArphrdByName = map[string]uint16{
	"netrom":             syscall.ARPHRD_NETROM,
	"ether":              syscall.ARPHRD_ETHER,
	"eether":             syscall.ARPHRD_EETHER,
	"ax25":               syscall.ARPHRD_AX25,
	"pronet":             syscall.ARPHRD_PRONET,
	"chaos":              syscall.ARPHRD_CHAOS,
	"ieee802":            syscall.ARPHRD_IEEE802,
	"arcnet":             syscall.ARPHRD_ARCNET,
	"appletlk":           syscall.ARPHRD_APPLETLK,
	"dlci":               syscall.ARPHRD_DLCI,
	"atm":                syscall.ARPHRD_ATM,
	"metricom":           syscall.ARPHRD_METRICOM,
	"ieee1394":           syscall.ARPHRD_IEEE1394,
	"eui64":              syscall.ARPHRD_EUI64,
	"infiniband":         syscall.ARPHRD_INFINIBAND,
	"slip":               syscall.ARPHRD_SLIP,
	"cslip":              syscall.ARPHRD_CSLIP,
	"slip6":              syscall.ARPHRD_SLIP6,
	"cslip6":             syscall.ARPHRD_CSLIP6,
	"rsrvd":              syscall.ARPHRD_RSRVD,
	"adapt":              syscall.ARPHRD_ADAPT,
	"rose":               syscall.ARPHRD_ROSE,
	"x25":                syscall.ARPHRD_X25,
	"hwx25":              syscall.ARPHRD_HWX25,
	"ppp":                syscall.ARPHRD_PPP,
	"hdlc":               syscall.ARPHRD_HDLC,
	"lapb":               syscall.ARPHRD_LAPB,
	"ddcmp":              syscall.ARPHRD_DDCMP,
	"rawhdlc":            syscall.ARPHRD_RAWHDLC,
	"tunnel":             syscall.ARPHRD_TUNNEL,
	"tunnel6":            syscall.ARPHRD_TUNNEL6,
	"frad":               syscall.ARPHRD_FRAD,
	"skip":               syscall.ARPHRD_SKIP,
	"loopback":           syscall.ARPHRD_LOOPBACK,
	"localtlk":           syscall.ARPHRD_LOCALTLK,
	"fddi":               syscall.ARPHRD_FDDI,
	"bif":                syscall.ARPHRD_BIF,
	"sit":                syscall.ARPHRD_SIT,
	"ipddp":              syscall.ARPHRD_IPDDP,
	"ipgre":              syscall.ARPHRD_IPGRE,
	"pimreg":             syscall.ARPHRD_PIMREG,
	"hippi":              syscall.ARPHRD_HIPPI,
	"ash":                syscall.ARPHRD_ASH,
	"econet":             syscall.ARPHRD_ECONET,
	"irda":               syscall.ARPHRD_IRDA,
	"fcpp":               syscall.ARPHRD_FCPP,
	"fcal":               syscall.ARPHRD_FCAL,
	"fcpl":               syscall.ARPHRD_FCPL,
	"fcfabric":           syscall.ARPHRD_FCFABRIC,
	"ieee802-tr":         syscall.ARPHRD_IEEE802_TR,
	"ieee80211":          syscall.ARPHRD_IEEE80211,
	"ieee80211-prism":    syscall.ARPHRD_IEEE80211_PRISM,
	"ieee80211-radiotap": syscall.ARPHRD_IEEE80211_RADIOTAP,
	"ieee802154":         syscall.ARPHRD_IEEE802154,
	"ieee802154-phy":     syscall.ARPHRD_IEEE802154_PHY,
}

var ArphrdName = map[uint16]string{
	syscall.ARPHRD_NETROM:             "netrom",
	syscall.ARPHRD_ETHER:              "ether",
	syscall.ARPHRD_EETHER:             "eether",
	syscall.ARPHRD_AX25:               "ax25",
	syscall.ARPHRD_PRONET:             "pronet",
	syscall.ARPHRD_CHAOS:              "chaos",
	syscall.ARPHRD_IEEE802:            "ieee802",
	syscall.ARPHRD_ARCNET:             "arcnet",
	syscall.ARPHRD_APPLETLK:           "appletlk",
	syscall.ARPHRD_DLCI:               "dlci",
	syscall.ARPHRD_ATM:                "atm",
	syscall.ARPHRD_METRICOM:           "metricom",
	syscall.ARPHRD_IEEE1394:           "ieee1394",
	syscall.ARPHRD_EUI64:              "eui64",
	syscall.ARPHRD_INFINIBAND:         "infiniband",
	syscall.ARPHRD_SLIP:               "slip",
	syscall.ARPHRD_CSLIP:              "cslip",
	syscall.ARPHRD_SLIP6:              "slip6",
	syscall.ARPHRD_CSLIP6:             "cslip6",
	syscall.ARPHRD_RSRVD:              "rsrvd",
	syscall.ARPHRD_ADAPT:              "adapt",
	syscall.ARPHRD_ROSE:               "rose",
	syscall.ARPHRD_X25:                "x25",
	syscall.ARPHRD_HWX25:              "hwx25",
	syscall.ARPHRD_PPP:                "ppp",
	syscall.ARPHRD_HDLC:               "hdlc",
	syscall.ARPHRD_LAPB:               "lapb",
	syscall.ARPHRD_DDCMP:              "ddcmp",
	syscall.ARPHRD_RAWHDLC:            "rawhdlc",
	syscall.ARPHRD_TUNNEL:             "tunnel",
	syscall.ARPHRD_TUNNEL6:            "tunnel6",
	syscall.ARPHRD_FRAD:               "frad",
	syscall.ARPHRD_SKIP:               "skip",
	syscall.ARPHRD_LOOPBACK:           "loopback",
	syscall.ARPHRD_LOCALTLK:           "localtlk",
	syscall.ARPHRD_FDDI:               "fddi",
	syscall.ARPHRD_BIF:                "bif",
	syscall.ARPHRD_SIT:                "sit",
	syscall.ARPHRD_IPDDP:              "ipddp",
	syscall.ARPHRD_IPGRE:              "ipgre",
	syscall.ARPHRD_PIMREG:             "pimreg",
	syscall.ARPHRD_HIPPI:              "hippi",
	syscall.ARPHRD_ASH:                "ash",
	syscall.ARPHRD_ECONET:             "econet",
	syscall.ARPHRD_IRDA:               "irda",
	syscall.ARPHRD_FCPP:               "fcpp",
	syscall.ARPHRD_FCAL:               "fcal",
	syscall.ARPHRD_FCPL:               "fcpl",
	syscall.ARPHRD_FCFABRIC:           "fcfabric",
	syscall.ARPHRD_IEEE802_TR:         "ieee802-tr",
	syscall.ARPHRD_IEEE80211:          "ieee80211",
	syscall.ARPHRD_IEEE80211_PRISM:    "ieee80211-prism",
	syscall.ARPHRD_IEEE80211_RADIOTAP: "ieee80211-radiotap",
	syscall.ARPHRD_IEEE802154:         "ieee802154",
	syscall.ARPHRD_IEEE802154_PHY:     "ieee802154-phy",
}

func CompleteArphrd(s string) (list []string) {
	for k := range ArphrdByName {
		if len(s) == 0 || strings.HasPrefix(k, s) {
			list = append(list, k)
		}
	}
	if len(list) > 0 {
		sort.Strings(list)
	}
	return
}
