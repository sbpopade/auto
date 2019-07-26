// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

// The is the dynamic library for platinasystems fe1
package main

import (
	"github.com/platinasystems/fe1"
	"github.com/platinasystems/go/vnet"
	vnet_fe1 "github.com/platinasystems/go/vnet/platforms/fe1"
)

func Packages() []map[string]string { return fe1.Packages }

func AddPlatform(v *vnet.Vnet, pp *vnet_fe1.Platform) {
	fe1.AddPlatform(v, pp)
}
