// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package elog

import (
	"github.com/platinasystems/go/elib"

	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"syscall"
	"time"
)

type View struct {
	viewEvents
	currentBufferEvents []bufferEvent
	allBufferEvents     bufferEventVec
	name                string
	Times               viewTimes
	shared
}

func (v *View) SetName(name string) { v.name = name }
func (v *View) Name() string        { return v.name }

func (v *View) numEvents(all bool) (l uint) {
	if v.allBufferEvents != nil {
		es := v.currentBufferEvents
		if all {
			es = v.allBufferEvents
		}
		l = uint(len(es))
	} else {
		es := v.currentViewEvents
		if all {
			es = v.allViewEvents
		}
		l = uint(len(es))
	}
	return
}

func (v *View) getEvent(i uint, all bool) (h *eventHeader) {
	if v.allBufferEvents != nil {
		es := v.currentBufferEvents
		if all {
			es = v.allBufferEvents
		}
		return &es[i].eventHeader
	} else {
		es := v.currentViewEvents
		if all {
			es = v.allViewEvents
		}
		return &es[i].eventHeader
	}
}

func (v *View) NumEvents() uint           { return v.numEvents(false) }
func (v *View) Event(i uint) *eventHeader { return v.getEvent(i, false) }

func (v *View) EventLines(i uint) (s []string) {
	l := &Log{s: &v.shared}
	if v.currentBufferEvents != nil {
		return v.currentBufferEvents[i].lines(l)
	}
	return v.viewEventLines(l, i)
}
func (v *View) EventCaller(i uint) (c *CallerInfo) {
	e := v.Event(i)
	_, c = v.getCallerInfo(e.callerIndex)
	return
}

func (b *Buffer) NewView() (v *View) {
	v = &View{}

	v.shared.sharedHeader = b.shared.sharedHeader
	v.shared.stringTable.copyFrom(&b.shared.stringTable)
	v.shared.eventFilterShared.copyFrom(&b.shared.eventFilterShared)

	l := len(v.currentBufferEvents)
	cap := b.Cap()
	mask := b.capMask()
	v.allBufferEvents.Resize(uint(cap))
	i := int(b.lockIndex(true))
	if i >= cap {
		l += copy(v.allBufferEvents[l:], b.events[i&mask:])
	}
	l += copy(v.allBufferEvents[l:], b.events[0:i&mask])
	b.lockIndex(false)
	v.allBufferEvents = v.allBufferEvents[:l]
	v.currentBufferEvents = v.allBufferEvents

	// Event ordering is not guaranteed due to GetCaller().
	// So we sort events by time.
	sort.Slice(v.allBufferEvents, func(i, j int) bool {
		ei, ej := &v.allBufferEvents[i], &v.allBufferEvents[j]
		return ei.timestamp < ej.timestamp
	})

	v.getViewTimes()
	return
}

func NewView() *View { return DefaultBuffer.NewView() }

func (v *View) getAllEvent(i uint) (h *eventHeader) {
	if v.allBufferEvents != nil {
		return &v.allBufferEvents[i].eventHeader
	}
	return &v.allViewEvents[i].eventHeader
}

// Make subview with only events between elapsed times t0 and t1.
func (v *View) SubView(t0, t1 float64) (n uint) {
	l := int(v.numEvents(true))
	if t0 > t1 {
		t0, t1 = t1, t0
	}
	i0 := sort.Search(l, func(i int) bool {
		e := v.getEvent(uint(i), true)
		et := e.ElapsedTime(v)
		return et >= t0
	})
	i1 := sort.Search(l, func(i int) bool {
		e := v.getEvent(uint(i), true)
		et := e.ElapsedTime(v)
		return et > t1
	})
	if v.allBufferEvents != nil {
		v.currentBufferEvents = v.allBufferEvents[i0:i1]
	} else {
		v.currentViewEvents = v.allViewEvents[i0:i1]
	}
	v.doViewTimes(t0, t1)
	return v.NumEvents()
}
func (v *View) Reset() {
	v.currentBufferEvents = v.allBufferEvents
	v.currentViewEvents = v.allViewEvents
	v.Times.StartTime = v.StartTime.Truncate(1 * time.Second)
	v.getViewTimes()
}

func (v *View) getViewTimes() {
	if v.Times.StartTime.IsZero() {
		v.Times.StartTime = v.StartTime.Truncate(1 * time.Second)
	}
	if ne := v.NumEvents(); ne > 0 {
		e0, e1 := v.Event(0), v.Event(ne-1)
		t0 := e0.ElapsedTime(v)
		t1 := e1.ElapsedTime(v)
		v.doViewTimes(t0, t1)
	}
	return
}

type viewTimes struct {
	// Time of first event in view rounded down.
	StartTime time.Time
	// Time relative to start time rounded down/up to nearest units.
	MinElapsed, MaxElapsed float64
	// Interval between max - min elapsed time.
	Dt       float64
	Unit     float64
	UnitName string
}

func (v *View) doViewTimes(t0, t1 float64) {
	t := &v.Times

	var (
		unitName          string
		timeUnit, maxUnit float64
	)

	{
		dt := t1 - t0
		if dt < 0 {
			panic("t0 < t1")
		}
		logDt := float64(0)
		if dt > 0 {
			logDt = math.Floor(math.Log10(dt))
		}
		switch {
		case logDt < -6:
			unitName = "ns"
			timeUnit = 1e-9
			maxUnit = 1e3
		case logDt < -3:
			unitName = "μs"
			timeUnit = 1e-6
			maxUnit = 1e3
		case logDt < 0:
			unitName = "ms"
			timeUnit = 1e-3
			maxUnit = 1e3
		case dt < 60:
			unitName = "sec"
			timeUnit = 1
			maxUnit = 60
		case dt < 60*60:
			unitName = "min"
			timeUnit = 60
			maxUnit = 60
		case dt < 24*60*60:
			unitName = "hr"
			timeUnit = 60 * 60
			maxUnit = 24 * 60 * 60
		default:
			unitName = "day"
			timeUnit = 24 * 60 * 60
			maxUnit = 1e3
		}
	}

	t0 = timeUnit * math.Floor(t0/timeUnit)
	t1 = timeUnit * math.Ceil(t1/timeUnit)

	startTimeUnit := maxUnit * timeUnit
	if timeUnit >= 1 {
		startTimeUnit = 1
	}
	nt := t.StartTime.Add(time.Duration(t0 * 1e9)).Truncate(time.Duration(startTimeUnit * 1e9))
	dt := nt.Sub(t.StartTime).Seconds()
	t.StartTime = nt
	t0 -= dt
	t1 -= dt

	t.MinElapsed = t0
	t.MaxElapsed = t1
	t.Dt = t1 - t0
	t.Unit = timeUnit
	t.UnitName = unitName
	return
}

func (v *View) print(w io.Writer, verbose bool, filteredEvents []uint, showFilteredEvents bool) {
	type row struct {
		Time  string `format:"%-30s"`
		Data  string `format:"%s" align:"left" width:"60"`
		Delta string `format:"%s" align:"left" width:"9"`
		Path  string `format:"%s" align:"left" width:"30"`
	}
	colMap := map[string]bool{
		"Delta": verbose,
		"Path":  verbose,
	}
	ne := v.NumEvents()
	if showFilteredEvents {
		ne = uint(len(filteredEvents))
	}
	rows := make([]row, 0, ne)
	lastTime := 0.
	for i := uint(0); i < ne; i++ {
		ei := i
		if showFilteredEvents {
			ei = filteredEvents[i]
		}
		e := v.Event(ei)
		t, delta := v.ElapsedTime(e), 0.
		if i > 0 {
			delta = t - lastTime
		}
		lastTime = t
		lines := v.EventLines(ei)
		for j := range lines {
			if lines[j] == "" {
				continue
			}
			indent := ""
			if j > 0 {
				indent = "  "
			}
			r := row{
				Data: indent + lines[j],
			}
			if j == 0 {
				r.Time = e.timeString(&v.shared)
				r.Delta = fmt.Sprintf("%8.6f", delta)
				_, c := v.getCallerInfo(e.callerIndex)
				r.Path = c.Name
			}
			rows = append(rows, r)
		}
	}
	elib.Tabulate(rows).WriteCols(w, colMap)
}

func (v *View) Print(w io.Writer, verbose bool)                  { v.print(w, verbose, nil, false) }
func (v *View) PrintEvents(w io.Writer, es []uint, verbose bool) { v.print(w, verbose, es, true) }
func (b *Buffer) Print(w io.Writer, detail bool)                 { b.NewView().Print(w, detail) }

// Dump log on SIGUP.
func (b *Buffer) PrintOnHangupSignal(w io.Writer, detail bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	for {
		<-c
		v := b.NewView()
		v.Print(w, detail)
	}
}
func PrintOnHangupSignal(w io.Writer, detail bool) { DefaultBuffer.PrintOnHangupSignal(w, detail) }

func (b *Buffer) SaveOnHangupSignal(file string) {
	b.panicSaveFile = file
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	for {
		<-c
		v := b.NewView()
		if err := v.SaveFile(file); err != nil {
			panic(err)
		}
	}
}
func SaveOnHangupSignal(file string) { DefaultBuffer.SaveOnHangupSignal(file) }

// Panic adds error to log for given buffer and saves log file (if configured).
func (b *Buffer) Panic(err interface{}) {
	if !Enabled() {
		return
	}
	F("panic %s", err)
	if file := b.panicSaveFile; file != "" {
		v := b.NewView()
		v.SaveFile(file)
	}
}

// Panic adds error to log and saves log file (if configured).
func Panic(err interface{}) { DefaultBuffer.Panic(err) }

func (c *CallerInfo) match(re *regexp.Regexp) bool {
	return re.MatchString(c.Name)
}

type EventMatch struct {
	Events []uint
}

func (v *View) EventsMatching(matching string, events0 []uint) (events []uint, err error) {
	events = events0
	if events != nil {
		events = events[:0]
	}
	var re *regexp.Regexp
	if re, err = regexp.Compile(matching); err != nil {
		return
	}
	for i := uint(0); i < v.NumEvents(); i++ {
		c := v.EventCaller(i)
		if c.match(re) {
			events = append(events, i)
		}
	}
	return
}

type viewEvent struct {
	eventHeader

	// View buffer b[lo:hi] contains encoded format and arguments.
	lo, hi uint32
}

type viewEvents struct {
	currentViewEvents []viewEvent
	allViewEvents     []viewEvent
	b                 elib.ByteVec
	args              []interface{}
}

func (v *viewEvents) viewEventLines(l *Log, ei uint) []string {
	e := &v.currentViewEvents[ei]
	b := v.b[e.lo:e.hi]
	i := 0
	if l.l != nil {
		l.l = l.l[:0]
	}
	for i < len(b) {
		if v.args != nil {
			v.args = v.args[:0]
		}
		x, n := binary.Uvarint(b[i:])
		i += n
		format := l.s.GetString(StringRef(x))

		for {
			var (
				a    interface{}
				kind byte
			)
			if a, kind, i = l.s.decodeArg(b, i); kind == fmtEnd {
				l.sprintf(format, v.args...)
				break
			} else {
				v.args = append(v.args, a)
			}
		}
	}
	return l.l
}

func (v *viewEvents) convertBufferEvent(l *Log, e *bufferEvent) {
	// Grab string table lock since encode may add to string table.
	l.s.fmtMu.Lock()
	defer l.s.fmtMu.Unlock()
	lo := v.b.Len()
	i := lo
	l.f = func(format string, args ...interface{}) {
		i0 := i
		_, i = fmtEncode(l.s, &v.b, i, true, nil, StringRefNil, format, args)
		if elib.Debug {
			fmtDecode(l.s, v.b[i0:])
		}
	}
	r := l.s.callers[e.callerIndex]
	e.format(r, l)
	var ve viewEvent
	ve.eventHeader = e.eventHeader
	ve.lo = uint32(lo)
	ve.hi = uint32(i)
	v.allViewEvents = append(v.allViewEvents, ve)
}

func (v *View) convertBufferEvents() {
	l := &Log{s: &v.shared}
	for i := range v.allBufferEvents {
		v.convertBufferEvent(l, &v.allBufferEvents[i])
	}
	v.allBufferEvents = nil
	v.currentBufferEvents = nil
	v.currentViewEvents = v.allViewEvents
	return
}
