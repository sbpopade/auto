// Copyright © 2015-2017 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package slice

import (
	"regexp"
	"testing"
	"time"

	"github.com/platinasystems/go/internal/test"
	"github.com/platinasystems/go/internal/test/docker"
)

type slice struct {
	docker.Docket
}

var Suite = test.Suite{
	Name: "slice",
	Tests: test.Tests{
		&slice{
			docker.Docket{
				Name: "vlan",
				Tmpl: "testdata/net/slice/vlan/conf.yaml.tmpl",
			},
		},
	},
}

func (slice *slice) Test(t *testing.T) {
	slice.Docket.Tests = test.Tests{
		&test.Unit{"check connectivity", slice.checkConnectivity},
		&test.Unit{"check FRR", slice.checkFrr},
		&test.Unit{"check routes", slice.checkRoutes},
		&test.Unit{"check interconnectivity",
			slice.checkInterConnectivity},
		&test.Unit{"check isolation", slice.checkIsolation},
		&test.Unit{"check punt stress", slice.checkStress},
		&test.Unit{"check connectivity2", slice.checkConnectivity},
		&test.Unit{"check routes2", slice.checkRoutes},
		&test.Unit{"check interconnectivity2",
			slice.checkInterConnectivity},
		&test.Unit{"check pci punt stress", slice.checkStressPci},
		&test.Unit{"check connectivity3", slice.checkConnectivity},
		&test.Unit{"check routes3", slice.checkRoutes},
		&test.Unit{"check interconnectivity3",
			slice.checkInterConnectivity},
	}
	slice.Docket.Test(t)
}

func (slice *slice) checkConnectivity(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "10.1.0.2"},
		{"RA-1", "10.1.0.1"},
		{"RA-1", "10.2.0.3"},
		{"RA-2", "10.2.0.2"},
		{"RA-2", "10.3.0.4"},
		{"CA-2", "10.3.0.3"},
		{"CB-1", "10.1.0.2"},
		{"RB-1", "10.1.0.1"},
		{"RB-1", "10.2.0.3"},
		{"RB-2", "10.2.0.2"},
		{"RB-2", "10.3.0.4"},
		{"CB-2", "10.3.0.3"},
	} {
		err := slice.PingCmd(t, x.hostname, x.target)
		assert.Nil(err)
		assert.Program(test.Self{},
			"vnet", "show", "ip", "fib", "table", x.hostname)
	}
}

func (slice *slice) checkFrr(t *testing.T) {
	assert := test.Assert{t}
	time.Sleep(1 * time.Second)
	for _, r := range slice.Routers {
		assert.Comment("Checking FRR on", r.Hostname)
		out, err := slice.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.True(regexp.MustCompile(".*ospfd.*").MatchString(out))
		assert.True(regexp.MustCompile(".*zebra.*").MatchString(out))
	}
}

func (slice *slice) checkRoutes(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		route    string
	}{
		{"CA-1", "10.3.0.0/24"},
		{"CA-2", "10.1.0.0/24"},
		{"CB-1", "10.3.0.0/24"},
		{"CB-2", "10.1.0.0/24"},
	} {
		found := false
		timeout := 120
		for i := timeout; i > 0; i-- {
			out, err := slice.ExecCmd(t, x.hostname,
				"ip", "route", "show", x.route)
			assert.Nil(err)
			if !assert.MatchNonFatal(out, x.route) {
				time.Sleep(1 * time.Second)
			} else {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("No ospf route for %v: %v", x.hostname, x.route)
		}
	}
}

func (slice *slice) checkInterConnectivity(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "10.3.0.4"}, // In slice A ping from CA-1 to CA-2
		{"CB-1", "10.3.0.4"}, // In slice B ping from CB-1 to CB-2
		{"CA-2", "10.1.0.1"}, // In slice A ping from CA-2 to CA-1
		{"CB-2", "10.1.0.1"}, // In slice B ping from CB-2 to CB-1

	} {
		err := slice.PingCmd(t, x.hostname, x.target)
		assert.Nil(err)
		assert.Program(test.Self{},
			"vnet", "show", "ip", "fib", "table", x.hostname)
	}
}

func (slice *slice) checkIsolation(t *testing.T) {
	assert := test.Assert{t}

	// break slice B connectivity does not affect slice A
	r, err := docker.FindHost(slice.Config, "RB-2")
	assert.Nil(err)

	for _, i := range r.Intfs {
		var intf string
		if i.Vlan != "" {
			intf = i.Name + "." + i.Vlan
		} else {
			intf = i.Name
		}
		_, err := slice.ExecCmd(t, r.Hostname,
			"ip", "link", "set", "down", intf)
		assert.Nil(err)
	}
	// how do I do an anti match???
	assert.Program(test.Self{},
		"vnet", "show", "ip", "fib", "table", "RB-2")

	assert.Comment("Verify that slice B is broken")
	_, err = slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.3.0.4")
	assert.NonNil(err)

	assert.Comment("Verify that slice A is not affected")
	_, err = slice.ExecCmd(t, "CA-1", "ping", "-c1", "10.3.0.4")
	assert.Nil(err)
	assert.Program(regexp.MustCompile("10.3.0.0/24"),
		test.Self{},
		"vnet", "show", "ip", "fib", "table", "RA-2")

	// bring RB-2 interfaces back up
	for _, i := range r.Intfs {
		var intf string
		if i.Vlan != "" {
			intf = i.Name + "." + i.Vlan
		} else {
			intf = i.Name
		}
		_, err := slice.ExecCmd(t, r.Hostname,
			"ip", "link", "set", "up", intf)
		assert.Nil(err)
	}

	// break slice A connectivity does not affect slice B
	r, err = docker.FindHost(slice.Config, "RA-2")
	assert.Nil(err)

	for _, i := range r.Intfs {
		var intf string
		if i.Vlan != "" {
			intf = i.Name + "." + i.Vlan
		} else {
			intf = i.Name
		}
		_, err := slice.ExecCmd(t, r.Hostname,
			"ip", "link", "set", "down", intf)
		assert.Nil(err)
	}
	// how do I do an anti match???
	assert.Program(test.Self{},
		"vnet", "show", "ip", "fib", "table", "RA-2")

	assert.Comment("Verify that slice A is broken")
	_, err = slice.ExecCmd(t, "CA-1", "ping", "-c1", "10.3.0.4")
	assert.NonNil(err)

	ok := false
	assert.Comment("Verify that slice B is not affected")
	timeout := 120
	for i := timeout; i > 0; i-- {
		out, _ := slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.3.0.4")
		if !assert.MatchNonFatal(out, "1 packets received") {
			time.Sleep(1 * time.Second)
		} else {
			ok = true
			break
		}
	}
	if !ok {
		t.Error("Slice B ping failed")
	}
	assert.Program(regexp.MustCompile("10.3.0.0/24"),
		test.Self{},
		"vnet", "show", "ip", "fib", "table", "RB-2")

	// bring RA-1 interfaces back up
	for _, i := range r.Intfs {
		var intf string
		if i.Vlan != "" {
			intf = i.Name + "." + i.Vlan
		} else {
			intf = i.Name
		}
		_, err := slice.ExecCmd(t, r.Hostname,
			"ip", "link", "set", "up", intf)
		assert.Nil(err)
	}

}

func (slice *slice) checkStress(t *testing.T) {
	assert := test.Assert{t}

	assert.Comment("stress with hping3")

	duration := []string{"1", "10", "30", "60"}

	ok := false
	timeout := 120
	for i := timeout; i > 0; i-- {
		out, _ := slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.3.0.4")
		if !assert.MatchNonFatal(out, "1 packets received") {
			time.Sleep(1 * time.Second)
		} else {
			ok = true
			assert.Comment("ping ok before stress")
			break
		}
	}
	if !ok {
		t.Error("ping failing before stress test")
	}

	for _, to := range duration {
		assert.Comment("stress for", to)
		_, err := slice.ExecCmd(t, "CB-1",
			"timeout", to,
			"hping3", "--icmp", "--flood", "-q", "10.3.0.4")
		assert.Comment("verfy can still ping neighbor")
		_, err = slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.1.0.2")
		assert.Nil(err)
	}
}

func (slice *slice) checkStressPci(t *testing.T) {
	assert := test.Assert{t}

	assert.Comment("stress with hping3 with ttl=1")

	duration := []string{"1", "10", "30", "60"}

	ok := false
	timeout := 120
	for i := timeout; i > 0; i-- {
		out, _ := slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.3.0.4")
		if !assert.MatchNonFatal(out, "1 packets received") {
			time.Sleep(1 * time.Second)
		} else {
			ok = true
			assert.Comment("ping ok before stress")
			break
		}
	}
	if !ok {
		t.Error("ping failing before stress test")
	}

	for _, to := range duration {
		assert.Comment("stress for", to)
		_, err := slice.ExecCmd(t, "CB-1",
			"timeout", to,
			"hping3", "--icmp", "--flood", "-q", "-t", "1",
			"10.3.0.4")
		assert.Comment("verfy can still ping neighbor")
		_, err = slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.1.0.2")
		assert.Nil(err)
	}
}
