// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package nl

import "sync/atomic"

var seq uint32

func Seq() uint32 {
	return atomic.AddUint32(&seq, 1)
}
