// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package resize

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/platinasystems/go/goes/cmd"
	"github.com/platinasystems/go/goes/lang"
)

type Command struct{}

func (Command) String() string { return "resize" }
func (Command) Usage() string  { return "resize" }

func (Command) Apropos() lang.Alt {
	return lang.Alt{
		lang.EnUS: "Resize per ROWS, COLUMNS and [XPIXELS, YPIXELS]",
	}
}

func (Command) Kind() cmd.Kind { return cmd.DontFork | cmd.CantPipe }

func (Command) Main(args ...string) error {
	var (
		rcxy    struct{ Row, Col, X, Y uint16 }
		mustset bool
		err     error
	)
	if len(args) != 0 {
		return fmt.Errorf("%v: unexpected", args)
	}
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdout),
		syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&rcxy)))
	if e < 0 {
		return fmt.Errorf("TIOCGWINSZ: %v", e)
	}
	for _, dimension := range []struct {
		name string
		rcxy *uint16
		init uint16
	}{
		{"ROWS", &rcxy.Row, 24},
		{"COLUMNS", &rcxy.Col, 80},
		{"XPIXELS", &rcxy.X, 0},
		{"YPIXELS", &rcxy.Y, 0},
	} {
		var u uint16
		env := os.Getenv(dimension.name)
		if len(env) != 0 {
			_, err = fmt.Sscan(env, &u)
			if err != nil {
				return fmt.Errorf("%s: %s: %v",
					dimension.name, env, err)
			}
		}
		if *(dimension.rcxy) == 0 {
			if u == 0 {
				if dimension.init != 0 {
					mustset = true
					*(dimension.rcxy) = dimension.init
					env = fmt.Sprint(dimension.init)
					os.Setenv(dimension.name, env)
				}
			} else {
				mustset = true
				*(dimension.rcxy) = u
			}
		} else if *(dimension.rcxy) != u {
			env = fmt.Sprint(*(dimension.rcxy))
			os.Setenv(dimension.name, env)
		}
	}
	if mustset {
		_, _, e := syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(syscall.Stdout),
			syscall.TIOCSWINSZ,
			uintptr(unsafe.Pointer(&rcxy)))
		if e < 0 {
			err = fmt.Errorf("TIOCSWINSZ: %v", e)
		}
	}
	return err
}
