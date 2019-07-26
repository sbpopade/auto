// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package docker

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/platinasystems/go/internal/test"
	"github.com/platinasystems/go/internal/test/netport"
)

type Docket struct {
	Name string
	Tmpl string
	*Config
	test.Tests
}

func (d *Docket) String() string { return d.Name }

func (d *Docket) Init(t *testing.T) {
	assert := test.Assert{t}
	assert.Helper()
	text, err := ioutil.ReadFile(d.Tmpl)
	assert.Nil(err)
	name := strings.TrimSuffix(d.Tmpl, ".tmpl")
	tmpl, err := template.New(name).Parse(string(text))
	assert.Nil(err)
	buf := new(bytes.Buffer)
	assert.Nil(tmpl.Execute(buf, netport.PortByNetPort))
	d.Config, err = LaunchContainers(t, buf.Bytes())
	assert.Nil(err)
}

func (d *Docket) Exit(t *testing.T) {
	if d.Config != nil {
		TearDownContainers(t, d.Config)
	}
}

func (d *Docket) ExecCmd(t *testing.T, ID string,
	cmd ...string) (string, error) {
	return ExecCmd(t, ID, d.Config, cmd)
}

func (d *Docket) PingCmd(t *testing.T, ID string, target string) error {
	return PingCmd(t, ID, d.Config, target)
}

func (d *Docket) Test(t *testing.T) {
	if *test.DryRun {
		fmt.Println(t.Name())
	} else {
		defer d.Exit(t)
		d.Init(t)
	}
	d.Tests.Test(t)
}
