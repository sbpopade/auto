// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vnet

import (
	"github.com/platinasystems/go/elib/cli"
	"github.com/platinasystems/go/elib/parse"
)

func (v *Vnet) CliAdd(c *cli.Command)                     { v.loop.CliAdd(c) }
func (v *Vnet) Logf(format string, args ...interface{})   { v.loop.Logf(format, args...) }
func (v *Vnet) Logln(args ...interface{})                 { v.loop.Logln(args...) }
func (v *Vnet) Fatalf(format string, args ...interface{}) { v.loop.Fatalf(format, args...) }

type cliListener struct {
	socketConfig string
	serverConfig cli.ServerConfig
	server       *cli.Server
}

func (l *cliListener) Parse(in *parse.Input) {
	for !in.End() {
		switch {
		case in.Parse("no-prompt"):
			l.serverConfig.DisablePrompt = true
		case in.Parse("enable-quit"):
			l.serverConfig.EnableQuit = true
		case in.Parse("socket %s", &l.socketConfig):
		default:
			in.ParseError()
		}
	}
}

type cliMain struct {
	Package
	v           *Vnet
	enableStdin bool
	listeners   []cliListener
}

func (v *Vnet) CliInit() {
	m := &v.cliMain
	m.v = v
	v.AddPackage("cli", m)
}

func (m *cliMain) Configure(in *parse.Input) {
	for !in.End() {
		var (
			l  cliListener
			li parse.Input
		)
		switch {
		case in.Parse("listen %v", &li) && li.Parse("%v", &l):
			m.listeners = append(m.listeners, l)
		case in.Parse("stdin"):
			m.enableStdin = true
		default:
			in.ParseError()
		}
	}
}

func (m *cliMain) Init() (err error) {
	m.v.loop.Cli.Prompt = "vnet# "
	m.v.loop.Cli.SetEventNode(&m.v.eventMain.eventNode)
	if m.enableStdin {
		m.v.loop.Cli.AddStdin()
	}
	for i := range m.listeners {
		l := &m.listeners[i]
		l.server, err = m.v.loop.Cli.AddServer(l.socketConfig, l.serverConfig)
		if err != nil {
			return
		}
	}
	m.v.loop.Cli.Start()
	return
}

func (m *cliMain) Exit() (err error) {
	for i := range m.listeners {
		l := &m.listeners[i]
		l.server.Close()
	}
	m.v.loop.Cli.Exit()
	return
}
