// Copyright © 2015-2018 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package grubd

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/platinasystems/go/goes"
	"github.com/platinasystems/go/goes/cmd"
	"github.com/platinasystems/go/goes/cmd/platina/mk1/bootc"
	"github.com/platinasystems/go/goes/lang"
)

type Command struct {
	g      *goes.Goes
	done   chan struct{}
	mounts []*bootMnt
}

type bootMnt struct {
	mnt     string
	err     error
	hasGrub bool
}

func (*Command) String() string { return "grubd" }

func (*Command) Usage() string {
	return "grubd [PATH]..."
}

func (*Command) Apropos() lang.Alt {
	return lang.Alt{
		lang.EnUS: "boot another operating system",
	}
}

func (c *Command) Close() error {
	close(c.done)
	return nil
}

func (c *Command) Goes(g *goes.Goes) { c.g = g }

func (*Command) Kind() cmd.Kind { return cmd.Daemon }

func (c *Command) Main(args ...string) (err error) {

	mp := "/mountd"
	if len(args) > 0 {
		mp = args[0]
	}
	c.done = make(chan struct{})

	done := make(chan *bootMnt, len(args))
	if c.mounts == nil {
		c.mounts = make([]*bootMnt, 0)
	}
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()

	var pccDone = false
	if kexec := bootc.Bootc(); len(kexec) > 1 {
		err := c.g.Main(kexec...)
		fmt.Println(err)
	}
	pccDone = true

	for {
		if pccDone {
			dirs, err := ioutil.ReadDir(mp)
			if err != nil {
				fmt.Printf("Error reading %s dir: %s", mp, err)
			}
			cnt := 0
			c.mounts = c.mounts[:0]
			for _, dir := range dirs {
				for _, sd := range []string{"", "/boot", "/d-i"} {
					m := &bootMnt{mnt: mp + "/" + dir.Name() + sd}
					c.mounts = append(c.mounts, m)
					go c.tryScanFiles(m, done)
					cnt++
				}
			}
			for i := 0; i < cnt; i++ {
				<-done
			}

			for _, m := range c.mounts {
				if m.hasGrub {
					args := []string{"grub", "--daemon"}
					args = append(args, m.mnt+"/grub/grub.cfg")
					fmt.Printf("%v\n", args)
					x := c.g.Fork(args...)
					x.Stdin = os.Stdin
					x.Stdout = os.Stdout
					x.Stderr = os.Stderr
					x.Dir = "/"
					err := x.Run()
					if err != nil {
						fmt.Printf("grub returned %s\n", err)
					}
				}
			}
		}

		select {
		case <-c.done:
			return nil
		case <-t.C:
		}
	}
}

func (*Command) tryScanFiles(m *bootMnt, done chan *bootMnt) {
	files, err := ioutil.ReadDir(m.mnt)
	if err != nil {
		m.err = err
		done <- m
		return
	}

	for _, file := range files {
		name := file.Name()
		if file.Mode().IsDir() && name == "grub" {
			if _, err := os.Stat(m.mnt + "/grub/grub.cfg"); err == nil {
				m.hasGrub = true
			} else {
				fmt.Printf("os.stat err %s\n", err)
			}
			continue
		}
	}
	done <- m
}
