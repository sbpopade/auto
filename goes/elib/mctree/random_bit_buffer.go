// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mctree

import (
	"math/rand"
)

type random_bit_buffer struct {
	// Number of random bits left in buffer.
	n uint
	b int64
}

func (x *random_bit_buffer) random_bit() (b int) {
	if x.n == 0 {
		x.n = 63
		x.b = int64(rand.Int63())
	}
	x.n--
	b = int((x.b >> x.n) & 1)
	return
}
