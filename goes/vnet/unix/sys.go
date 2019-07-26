// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unix

import (
	"syscall"
	"unsafe"
)

type iovec syscall.Iovec

//go:generate gentemplate -d Package=unix -id iovec -d VecType=iovecVec -d Type=iovec github.com/platinasystems/go/elib/vec.tmpl

func rwv(fd int, iov []iovec, isWrite bool) (n int, e syscall.Errno) {
	sc := syscall.SYS_READV
	if isWrite {
		sc = syscall.SYS_WRITEV
	}
	r0, _, e := syscall.Syscall(uintptr(sc), uintptr(fd), uintptr(unsafe.Pointer(&iov[0])), uintptr(len(iov)))
	n = int(r0)
	return
}

func readv(fd int, iov []iovec) (int, syscall.Errno)  { return rwv(fd, iov, false) }
func writev(fd int, iov []iovec) (int, syscall.Errno) { return rwv(fd, iov, true) }

func rwmsg(fd, flags int, msg *msghdr, isWrite bool) (n int, e syscall.Errno) {
	sc := syscall.SYS_RECVMSG
	if isWrite {
		sc = syscall.SYS_SENDMSG
	}
	r0, _, e := syscall.Syscall(uintptr(sc), uintptr(fd), uintptr(unsafe.Pointer(msg)), uintptr(flags))
	n = int(r0)
	return
}
func recvmsg(fd, flags int, msg *msghdr) (int, syscall.Errno) { return rwmsg(fd, flags, msg, false) }
func sendmsg(fd, flags int, msg *msghdr) (int, syscall.Errno) { return rwmsg(fd, flags, msg, true) }

type msghdr syscall.Msghdr

type mmsghdr struct {
	msg_hdr msghdr
	msg_len uint32
}

func rwmmsg(fd, flags int, msgs []mmsghdr, isWrite bool) (n int, e syscall.Errno) {
	sc := syscall.SYS_RECVMMSG
	if isWrite {
		// sc = syscall.SYS_SENDMMSG not defined for linux amd64
		sc = 307 // fixme amd64 linux specific
	}
	r0, _, e := syscall.Syscall6(uintptr(sc), uintptr(fd), uintptr(unsafe.Pointer(&msgs[0])), uintptr(len(msgs)), uintptr(flags),
		uintptr(0), uintptr(0))
	n = int(r0)
	return
}
func recvmmsg(fd, flags int, msgs []mmsghdr) (int, syscall.Errno) {
	return rwmmsg(fd, flags, msgs, false)
}
func sendmmsg(fd, flags int, msgs []mmsghdr) (int, syscall.Errno) {
	return rwmmsg(fd, flags, msgs, true)
}

func (h *msghdr) set(a *syscall.RawSockaddrLinklayer, iovs []iovec, flags int) {
	h.Name = (*byte)(unsafe.Pointer(a))
	h.Namelen = syscall.SizeofSockaddrLinklayer
	h.Iov = (*syscall.Iovec)(&iovs[0])
	h.Iovlen = uint64(len(iovs))
	h.Flags = int32(flags)
}
