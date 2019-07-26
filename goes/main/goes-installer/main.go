// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

// +build amd64 arm

// This is the goes installer that extracts a zipped machine.
package main

import (
	"fmt"
	"os"

	"github.com/platinasystems/go/internal/machine"
)

const name = "installer"

func main() {
	var ecode int
	machine.Name = name
	if err := Goes.Main(os.Args...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		ecode = 1
	}
	os.Exit(ecode)
}
