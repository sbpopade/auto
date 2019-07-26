// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package n

import (
	"fmt"

	"github.com/platinasystems/go/goes"
	"github.com/platinasystems/go/goes/lang"
	"github.com/platinasystems/go/internal/netns"
)

type Command struct {
	Name string
	g    *goes.Goes
}

func (c *Command) String() string { return c.Name }

func (c *Command) Usage() string {
	return fmt.Sprintf("ip %s NAME OBJECT [ COMMAND [ ARGS ]...]", c)
}

func (*Command) Apropos() lang.Alt {
	return lang.Alt{
		lang.EnUS: "run ip command in namespace",
	}
}

func (*Command) Man() lang.Alt {
	return lang.Alt{
		lang.EnUS: `
SEE ALSO
	man ip || ip -man`,
	}
}

func (c *Command) Goes(g *goes.Goes) { c.g = g }

func (c *Command) Main(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing NAME")
	}
	if err := netns.Switch(args[0]); err != nil {
		return err
	}
	return c.g.Main(args[1:]...)
}

func (c *Command) Complete(args ...string) (list []string) {
	switch len(args) {
	case 0:
		list = netns.CompleteName("")
	case 1:
		list = netns.CompleteName(args[0])
	default:
		list = c.g.Complete(args[1:]...)
	}
	return
}
