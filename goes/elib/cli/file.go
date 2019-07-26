// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"github.com/platinasystems/go/elib/iomux"

	"encoding/binary"
	"fmt"
	"strings"
	"syscall"
)

func (c *File) ReadReady() (err error) {
	err = c.FileReadWriteCloser.ReadReady()
	if l := len(c.Read(0)); err == nil && l > 0 {
		c.main.RxReady(c)
	}
	return
}

func (c *File) WriteReady() (err error) {
	if err = c.FileReadWriteCloser.WriteReady(); err == nil {
		if c.closeAfterTxFlush && !c.WriteAvailable() {
			c.closeAfterTxFlush = false
			c.Close()
		}
	}
	return
}

// Either close immediately or wait until tx buffer is empty to close.
func (c *File) close() {
	if c.WriteAvailable() {
		c.closeAfterTxFlush = true
	} else {
		c.Close()
	}
}

func (c *File) writePrompt() {
	if l := len(c.main.Prompt); !c.DisablePrompt && l > 0 {
		c.Write([]byte(c.main.Prompt))
	}
}

func (c *File) RxReady() (err error) {
	for {
		b := c.Read(0)
		nl := strings.Index(string(b), "\n")
		if nl == -1 {
			return
		}
		end := nl
		if end > 0 && b[end-1] == '\r' {
			end--
		}
		if end > 0 {
			err = c.main.Exec(c, strings.NewReader(string(b[:end])))
			if err != nil {
				if s := err.Error(); len(s) > 0 {
					fmt.Fprintf(c, "%s\n", s)
				}
			}
			c.markEndOfOutput()
			if err == ErrQuit {
				// Quit is only quit from stdin; otherwise just close file.
				if !c.EnableQuit {
					c.close()
					err = nil
				}
				return
			}
			// The only error we bubble to callers is ErrQuit
			err = nil
		}
		c.writePrompt()
		// Advance read buffer.
		c.Read(nl + 1)
	}
}

func (c *Main) AddFile(f iomux.FileReadWriteCloser, cf ServerConfig) {
	i := c.FilePool.GetIndex()
	x := &c.Files[i]
	*x = File{
		main:                c,
		FileReadWriteCloser: f,
		poolIndex:           fileIndex(i),
	}
	x.ServerConfig = cf
	iomux.Add(x)
	x.writePrompt()
}

func (c *Main) Exit() {
	c.FilePool.ForeachIndex(func(i uint) {
		f := &c.Files[i]
		f.Close()
		c.FilePool.PutIndex(i)
	})
}

func (c *Main) AddStdin() {
	c.AddFile(iomux.NewFileBuf(syscall.Stdin, "stdin"), ServerConfig{EnableQuit: true})
}

func (f *File) isStdin() bool {
	if f, ok := f.FileReadWriteCloser.(*iomux.FileBuf); ok {
		return f.Fd == syscall.Stdin
	}
	return false
}

func (f *File) markEndOfOutput() {
	if f.DisablePrompt {
		f.Write([]byte{})
	}
}

func (f *File) Write(p []byte) (n int, err error) {
	if f.DisablePrompt {
		var tmp [4]byte
		binary.BigEndian.PutUint32(tmp[:], uint32(len(p)))
		n, err = f.FileReadWriteCloser.Write(tmp[:])
		if n != len(tmp) {
			return
		}
	}
	n, err = f.FileReadWriteCloser.Write(p)
	return
}

func (m *Main) Write(p []byte) (n int, err error) {
	if len(m.FilePool.Files) == 0 {
		n, err = syscall.Write(syscall.Stderr, p)
		return
	}

	for i := range m.FilePool.Files {
		if !m.FilePool.IsFree(uint(i)) {
			n, err = m.FilePool.Files[i].Write(p)
			return
		}
	}
	return
}

func (m *Main) Start() {
	for _, c := range builtins {
		m.AddCommand(c)
	}

	for _, cmd := range m.allCmds {
		if l, ok := cmd.(LoopStarter); ok {
			l.CliLoopStart(m)
		}
	}
}

func (c *Main) End() {
	// Restore Stdin to blocking on exit.
	for i := range c.Files {
		if !c.FilePool.IsFree(uint(i)) && c.Files[i].isStdin() {
			syscall.SetNonblock(syscall.Stdin, false)
		}
	}
}

func (c *Main) Loop() {
	rxReady := make(chan fileIndex)
	c.RxReady = func(c *File) {
		rxReady <- c.poolIndex
	}
	c.Start()
	defer c.End()
	for {
		i := <-rxReady
		if err := c.Files[i].RxReady(); err == ErrQuit {
			break
		}
	}
}
