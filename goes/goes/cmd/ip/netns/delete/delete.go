// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package delete

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/platinasystems/go/goes/cmd/ip/internal/options"
	"github.com/platinasystems/go/goes/lang"
	"github.com/platinasystems/go/internal/netns"
	"github.com/platinasystems/go/internal/nl/rtnl"
)

type Command struct{}

func (Command) String() string { return "delete" }

func (Command) Usage() string {
	return `ip netns delete [ -all | NETNSNAME ]`
}

func (Command) Apropos() lang.Alt {
	return lang.Alt{
		lang.EnUS: "remove network namespace",
	}

}

func (Command) Man() lang.Alt {
	return lang.Alt{
		lang.EnUS: `
SEE ALSO
	ip man netns || ip netns -man
	man ip || ip -man`,
	}
}

func (Command) Main(args ...string) error {
	opt, args := options.New(args)

	switch {
	case opt.Flags.ByName["-all"]:
		if len(args) > 0 {
			return fmt.Errorf("%v: unexpected", args)
		}
		varRunNetns, err := ioutil.ReadDir(rtnl.VarRunNetns)
		if err != nil {
			return err
		}
		for _, fi := range varRunNetns {
			if err := del(fi.Name()); err != nil {
				return err
			}
		}
	case len(args) == 0:
		return fmt.Errorf("NETNSNAME: missing")
	case len(args) == 1:
		return del(args[0])
	default:
		return fmt.Errorf("%v: unexpected", args[1:])
	}
	return nil
}

func (Command) Complete(args ...string) (list []string) {
	var larg string
	n := len(args)
	if n > 0 {
		larg = args[n-1]
	}
	for _, name := range []string{
		"-all",
	} {
		if strings.HasPrefix(name, larg) {
			list = append(list, name)
		}
	}
	if len(list) == 0 {
		list = netns.CompleteName(larg)
	}
	return
}

func del(name string) error {
	fn := filepath.Join(rtnl.VarRunNetns, name)
	if err := syscall.Unmount(fn, syscall.MNT_DETACH); err != nil {
		return err
	}
	return syscall.Unlink(fn)
}
