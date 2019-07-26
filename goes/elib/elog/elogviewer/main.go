package main

import (
	"github.com/platinasystems/go/elib"
	"github.com/platinasystems/go/elib/elog"
	"github.com/platinasystems/go/elib/elog/elogview"

	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"
)

type color uint32

var colorNames = [...]string{
	0: "dark blue",
	1: "light blue",
	2: "green",
	3: "yellow",
	4: "orange",
	5: "red",
}

func (c color) String() string { return colorNames[c] }

type ev struct {
	i     uint32
	color color
}

func (e *ev) Elog(l *elog.Log) { l.Logf("%s %d", e.color, e.i) }

func main() {
	var (
		n_events     uint
		delay        float64
		random_delay bool
		save, load   string
		useFmt       bool
		dump         bool
	)
	if elib.Debug {
		flag.Float64Var(&delay, "delay", 0, "delay in seconds between events or max delay for random delays.")
		flag.UintVar(&n_events, "events", 10, "number of test events to add")
		flag.BoolVar(&random_delay, "random", false, "randomize delays")
		flag.StringVar(&save, "save", "", "save log to file")
		flag.BoolVar(&useFmt, "fmt", false, "use elog.F* formatted functions")
	}
	flag.StringVar(&load, "load", "", "load log from file")
	flag.BoolVar(&dump, "dump", false, "dump log to stdout")
	flag.Parse()

	if as := flag.Args(); len(as) == 1 {
		load = as[0]
	}

	if !elib.Debug && len(load) == 0 {
		fmt.Println("expecting event log file to load")
		return
	}

	var v *elog.View

	if load != "" {
		if f, err := os.OpenFile(load, os.O_RDONLY, 0); err != nil {
			log.Fatal(err)
		} else {
			defer f.Close()
			var r elog.View
			if err = r.Restore(f); err != nil {
				log.Fatal(err)
			}
			v = &r
		}
	} else {
		elog.DefaultBuffer.Resize(n_events)
		elog.Enable(true)
		var ms [2]runtime.MemStats
		runtime.ReadMemStats(&ms[0])
		nColor := len(colorNames)
		fmts := make([]string, nColor)
		for i := range fmts {
			fmts[i] = colorNames[i] + " %d"
		}
		for i := uint64(0); i < uint64(n_events); i++ {
			color := color(i % uint64(len(colorNames)))
			if useFmt {
				fmt := fmts[color]
				switch color {
				case 0:
					elog.F1u(fmt, i)
				case 1:
					elog.F1u(fmt, i)
				case 2:
					elog.F1u(fmt, i)
				case 3:
					elog.F1u(fmt, i)
				case 4:
					elog.F1u(fmt, i)
				case 5:
					elog.F1u(fmt, i)
				}
			} else {
				e := ev{color: color, i: uint32(i)}
				switch color {
				case 0:
					elog.Add(&e)
				case 1:
					elog.Add(&e)
				case 2:
					elog.Add(&e)
				case 3:
					elog.Add(&e)
				case 4:
					elog.Add(&e)
				case 5:
					elog.Add(&e)
				}
			}
			if delay > 0 {
				d := delay
				if random_delay {
					d = rand.Float64() * delay
				}
				time.Sleep(time.Duration(1e9 * d))
			}
		}
		runtime.ReadMemStats(&ms[1])
		fmt.Printf("mallocs %d gcs %d\n", ms[1].Mallocs-ms[0].Mallocs, ms[1].NumGC-ms[0].NumGC)
		v = elog.NewView()
	}
	if save != "" {
		if f, err := os.OpenFile(save, os.O_CREATE|os.O_RDWR, 0666); err != nil {
			panic(err)
		} else {
			defer f.Close()
			v.SetName(save)
			if err = v.Save(f); err != nil {
				panic(err)
			}
		}
	}
	if dump {
		v.Print(os.Stdout, false)
	} else {
		cf := elogview.Config{
			Width:              1200,
			Height:             750,
			EnableKeyboardQuit: true,
		}
		elogview.View(v, cf)
	}
}
