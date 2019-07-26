// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

// +build plugin

package fe1

import (
	"plugin"

	"github.com/platinasystems/go/vnet"
	fe1_platform "github.com/platinasystems/go/vnet/platforms/fe1"
)

const FileName = "/usr/lib/goes/fe1.so"

var lib *plugin.Plugin

func lookup(name string) plugin.Symbol {
	if lib == nil {
		var err error
		lib, err = plugin.Open(FileName)
		if err != nil {
			panic(err)
		}
	}
	sym, err := lib.Lookup(name)
	if err != nil {
		panic(err)
	}
	return sym
}

func Packages() []map[string]string {
	return lookup("Packages").(func() []map[string]string)()
}

func Init(v *vnet.Vnet, p *fe1_platform.Platform) {
	lookup("Init").(func(v *vnet.Vnet, p *fe1_platform.Platform))(v, p)
}

func AddPlatform(v *vnet.Vnet, p *fe1_platform.Platform) {
	lookup("AddPlatform").(func(v *vnet.Vnet, p *fe1_platform.Platform))(v, p)
}
