// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hw

import (
	"github.com/platinasystems/go/elib"
	"github.com/platinasystems/go/elib/cpu"
	"github.com/platinasystems/go/elib/elog"

	"fmt"
	"math"
	"reflect"
	"sync"
	"unsafe"
)

type BufferFlag uint32

const (
	NextValid, Log2NextValid BufferFlag = 1 << iota, iota
	Cloned, Log2Cloned
)

var bufferFlagStrings = [...]string{
	Log2NextValid: "next-valid",
	Log2Cloned:    "cloned",
}

func (f BufferFlag) String() string { return elib.FlagStringer(bufferFlagStrings[:], elib.Word(f)) }

type RefHeader struct {
	// 28 bits of offset; 4 bits of flags.
	offsetAndFlags uint32

	dataOffset uint16
	dataLen    uint16
}

const (
	RefBytes       = 16
	RefHeaderBytes = 1*4 + 2*2
	RefOpaqueBytes = RefBytes - RefHeaderBytes
)

type Ref struct {
	RefHeader

	// User opaque area.
	opaque [RefOpaqueBytes]byte
}

func (r *RefHeader) offset() uint32         { return r.offsetAndFlags &^ 0xf }
func (r *RefHeader) Buffer() unsafe.Pointer { return DmaGetPointer(uint(r.offset())) }
func (r *RefHeader) GetBuffer() *Buffer     { return (*Buffer)(r.Buffer()) }
func (r *RefHeader) DataOffset(o uint) unsafe.Pointer {
	return DmaGetPointer(uint(r.offset() + uint32(r.dataOffset) + uint32(o)))
}
func (r *RefHeader) Data() unsafe.Pointer { return r.DataOffset(0) }
func (r *RefHeader) DataPhys() uintptr    { return DmaPhysAddress(uintptr(r.Data())) }

func (r *RefHeader) Flags() BufferFlag         { return BufferFlag(r.offsetAndFlags & 0xf) }
func (r *RefHeader) NextValidFlag() BufferFlag { return BufferFlag(r.offsetAndFlags) & NextValid }
func (r *RefHeader) NextIsValid() bool         { return r.NextValidFlag() != 0 }
func (r *RefHeader) SetFlags(f BufferFlag)     { r.offsetAndFlags |= uint32(f) }
func (r *RefHeader) nextValidUint() uint       { return uint(1 & (r.offsetAndFlags >> Log2NextValid)) }

func RefFlag1(f BufferFlag, r0 *RefHeader) bool { return r0.offsetAndFlags&uint32(f) != 0 }
func RefFlag2(f BufferFlag, r0, r1 *RefHeader) bool {
	return (r0.offsetAndFlags|r1.offsetAndFlags)&uint32(f) != 0
}
func RefFlag4(f BufferFlag, r0, r1, r2, r3 *RefHeader) bool {
	return (r0.offsetAndFlags|r1.offsetAndFlags|r2.offsetAndFlags|r3.offsetAndFlags)&uint32(f) != 0
}
func refFlag1(f BufferFlag, r0 *Ref) bool { return RefFlag1(f, &r0.RefHeader) }
func refFlag4(f BufferFlag, r0, r1, r2, r3 *Ref) bool {
	return RefFlag4(f, &r0.RefHeader, &r1.RefHeader, &r2.RefHeader, &r3.RefHeader)
}

func (r *RefHeader) DataSliceOffsetLen(o, l uint) (b []byte) {
	var h reflect.SliceHeader
	h.Data = uintptr(r.DataOffset(o))
	h.Len = int(l) - int(o)
	h.Cap = h.Len
	b = *(*[]byte)(unsafe.Pointer(&h))
	return
}
func (r *RefHeader) DataSlice() (b []byte)             { return r.DataSliceOffsetLen(0, r.DataLen()) }
func (r *RefHeader) DataSliceOffset(o uint) (b []byte) { return r.DataSliceOffsetLen(o, r.DataLen()) }
func (r *RefHeader) DataLen() uint                     { return uint(r.dataLen) }
func (r *RefHeader) SetDataLen(l uint)                 { r.dataLen = uint16(l) }
func (r *RefHeader) Advance(i int) (oldDataOffset int) {
	oldDataOffset = int(r.dataOffset)
	r.dataOffset = uint16(oldDataOffset + i)
	r.dataLen = uint16(int(r.dataLen) - i)
	return
}
func (r *RefHeader) Restore(oldDataOffset int) {
	r.dataOffset = uint16(oldDataOffset)
	Δ := int(r.dataOffset) - oldDataOffset
	r.dataLen = uint16(int(r.dataLen) - Δ)
}

//go:generate gentemplate -d Package=hw -id Ref -d VecType=RefVec -d Type=Ref github.com/platinasystems/go/elib/vec.tmpl

const (
	// Cache aligned/sized space for buffer header.
	BufferHeaderBytes = cpu.CacheLineBytes
	// Rewrite (prepend) area.
	BufferRewriteBytes = 64
	overheadBytes      = BufferHeaderBytes + BufferRewriteBytes
)

type BufferSave uint32

// Buffer header.
type BufferHeader struct {
	// Valid only if NextValid flag is set.
	nextRef RefHeader

	// Number of clones of this buffer.
	cloneCount uint32

	save BufferSave
}

func (h *BufferHeader) SetSave(x BufferSave) { h.save = x }
func (h *BufferHeader) GetSave() BufferSave  { return h.save }

func (r *RefHeader) NextRef() (x *RefHeader) {
	if r.Flags()&NextValid != 0 {
		x = &r.GetBuffer().nextRef
	}
	return
}

func LinkRefs(as, bs, cs *RefHeader, chain_len, n uint) {
	b, c := bs.slice(n), cs.slice(n)
	var a []Ref
	if as != nil {
		a = as.slice(n)
	}
	i, n_left := uint(0), n

	if a != nil {
		for n_left > 0 {
			ra0 := &a[i+0]
			rb0 := &b[i+0]
			rc0 := &c[i+0]
			a0 := ra0.GetBuffer()
			b0 := rb0.GetBuffer()
			a0.nextRef.SetFlags(NextValid)
			b0.nextRef = rc0.RefHeader
			n_left -= 1
			i += 1
		}
	} else {
		for n_left >= 4 {
			rb0, rb1, rb2, rb3 := &b[i+0], &b[i+1], &b[i+2], &b[i+3]
			rc0, rc1, rc2, rc3 := &c[i+0], &c[i+1], &c[i+2], &c[i+3]
			b0, b1, b2, b3 := rb0.GetBuffer(), rb1.GetBuffer(), rb2.GetBuffer(), rb3.GetBuffer()
			rb0.SetFlags(NextValid)
			rb1.SetFlags(NextValid)
			rb2.SetFlags(NextValid)
			rb3.SetFlags(NextValid)
			b0.nextRef = rc0.RefHeader
			b1.nextRef = rc1.RefHeader
			b2.nextRef = rc2.RefHeader
			b3.nextRef = rc3.RefHeader
			n_left -= 4
			i += 4
		}

		for n_left > 0 {
			rb0 := &b[i+0]
			rc0 := &c[i+0]
			b0 := rb0.GetBuffer()
			rb0.SetFlags(NextValid)
			b0.nextRef = rc0.RefHeader
			n_left -= 1
			i += 1
		}
	}
}

type Buffer struct {
	BufferHeader
	Opaque [BufferHeaderBytes - unsafe.Sizeof(BufferHeader{})]byte
}

type BufferPool struct {
	m *BufferMain

	// Index in bufferPools
	index uint32

	Name string

	ElogName elog.StringRef

	BufferTemplate

	// Mutually excludes allocate and free.
	mu sync.Mutex

	// References to buffers in this pool.
	refs RefVec

	// DMA memory chunks used by this pool.
	memChunkIDs []elib.Index

	// Number of bytes of dma memory allocated by this pool.
	DmaMemAllocBytes uint64

	freeNext freeNext
}

//go:generate gentemplate -d Package=hw -id bufferPools -d PoolType=bufferPools -d Type=*BufferPool -d Data=elts github.com/platinasystems/go/elib/pool.tmpl

type BufferState uint8

const (
	BufferUnknown BufferState = iota
	BufferKnownAllocated
	BufferKnownFree
)

var bufferStateStrings = [...]string{
	BufferUnknown:        "unknown",
	BufferKnownAllocated: "known-allocated",
	BufferKnownFree:      "known-free",
}

func (s BufferState) String() string { return elib.Stringer(bufferStateStrings[:], int(s)) }

const traceBuffers = elib.Debug

func (m *BufferMain) getTrace(offset uint32) (t *bufferTrace) {
	m.Lock()
	defer m.Unlock()
	if m.traceByOffset == nil {
		m.traceByOffset = make(map[uint32]*bufferTrace)
	}
	var ok bool
	if t, ok = m.traceByOffset[offset]; !ok {
		t = &bufferTrace{offset: offset}
		m.traceByOffset[offset] = t
	}
	return
}
func (m *BufferMain) putTrace(offset uint32) {
	if t, ok := m.traceByOffset[offset]; ok {
		t.reset()
	}
}

func (s BufferState) TraceBuffer(i int) string { return "set-state: " + s.String() }

func (p *BufferPool) setState(offset uint32, new BufferState) (t *bufferTrace, old BufferState) {
	t = p.m.getTrace(offset)
	old = t.state
	t.state = new
	if new == BufferKnownAllocated {
		t.reset()
	}
	t.events = append(t.events, bufferEvent{p: p.index, i: new})
	if elog.Enabled() {
		e := bufferElog{
			pool:   p.ElogName,
			offset: offset,
			old:    old,
			new:    new,
		}
		elog.Add(&e)
	}
	return
}

func (p *BufferPool) unsetState(offset uint32) {
	if traceBuffers {
		p.m.putTrace(offset)
	}
}

func (r *RefHeader) Trace(p *BufferPool, i BufferTracer, e int) {
	if !traceBuffers {
		return
	}
	t := p.m.getTrace(r.offset())
	t.events = append(t.events, bufferEvent{i: i, e: uint32(e), p: p.index})
}

func (t *bufferTrace) TraceString(m *BufferMain) (lines []string) {
	lines = append(lines, fmt.Sprintf("events for offset 0x%x:", t.offset))
	for i := range t.events {
		e := &t.events[i]
		p := m.bufferPools.elts[e.p]
		lines = append(lines, fmt.Sprintf("  pool %s: %s", p.Name, e.i.TraceBuffer(int(e.e))))
	}
	return
}

func (p *BufferPool) panicState(t *bufferTrace, got, want BufferState) {
	lines := t.TraceString(p.m)
	for i := range lines {
		fmt.Println(lines[i])
	}
	panic(fmt.Errorf("validate buffer offset 0x%x: want %s != got %s", t.offset, want, got))
}

func (p *BufferPool) ValidateRefs(h *RefHeader, want BufferState, n, stride uint) {
	if !traceBuffers {
		return
	}
	refs := h.slice(n)
	for i := uint(0); i < uint(len(refs)); i += stride {
		r := refs[i].RefHeader
		t := p.m.getTrace(r.offset())
		if got := t.state; got != want {
			p.panicState(t, got, want)
		}
	}
}

func (p *BufferPool) validateSetState(r RefHeader, set BufferState) {
	if !traceBuffers {
		return
	}
	t, got := p.setState(r.offset(), set)
	want := BufferKnownAllocated
	if set == BufferKnownAllocated {
		// Accept either known free or unknown (for initial allocation).
		want = BufferKnownFree
		if got == BufferUnknown {
			want = BufferUnknown
		}
	}
	if got != want {
		p.panicState(t, got, want)
	}
}

func (r *RefHeader) ValidateState(p *BufferPool, want BufferState) {
	if !traceBuffers {
		return
	}
	t := p.m.getTrace(r.offset())
	if got := t.state; got != want {
		p.panicState(t, got, want)
	}
}

func (p *BufferPool) validateSetStateRefs(r []Ref, set BufferState, stride uint) {
	if !traceBuffers {
		return
	}
	for i := uint(0); i < uint(len(r)); i += stride {
		p.validateSetState(r[i].RefHeader, set)
	}
}

// Method to over-ride to initialize refs for this buffer pool.
// This is used for example to set packet lengths, adjust packet fields, etc.
func (p *BufferPool) InitRefs(refs []Ref) {}

func isPrime(i uint) bool {
	max := uint(math.Sqrt(float64(i)))
	for j := uint(2); j <= max; j++ {
		if i%j == 0 {
			return false
		}
	}
	return true
}

// Size of a data buffer in given free list.
// Choose to be a prime number of cache lines to randomize addresses for better cache usage.
func (p *BufferPool) bufferSize() uint {
	nBytes := overheadBytes + p.Size
	nLines := nBytes / cpu.CacheLineBytes
	for !isPrime(nLines) {
		nLines++
	}
	return nLines * cpu.CacheLineBytes
}

type BufferTemplate struct {
	// Data size of buffers.
	Size uint

	sizeIncludingOverhead uint

	Ref
	Buffer

	// If non-nil buffers will be initialized with this data.
	Data []byte
}

func (t *BufferTemplate) SizeIncludingOverhead() uint { return t.sizeIncludingOverhead }

var DefaultBufferTemplate = &BufferTemplate{
	Size: 1 << 10,
	Ref:  Ref{RefHeader: RefHeader{dataOffset: BufferRewriteBytes}},
}

func (p *BufferPool) isInitialized() bool { return p.sizeIncludingOverhead != 0 }

func (p *BufferPool) Init() {
	t := &p.BufferTemplate
	if len(t.Data) > 0 {
		t.Ref.dataLen = uint16(len(t.Data))
	}
	// User does not get to choose buffer header.
	t.Buffer.BufferHeader = BufferHeader{save: t.Buffer.BufferHeader.save}
	p.Size = uint(elib.Word(p.Size).RoundCacheLine())
	p.sizeIncludingOverhead = p.bufferSize()
	p.Size = p.sizeIncludingOverhead - overheadBytes
}

type BufferTracer interface {
	TraceBuffer(e int) string
}

type bufferEvent struct {
	i BufferTracer
	e uint32
	p uint32
}

type bufferTrace struct {
	offset uint32
	// Current buffer state (known allocated or known free).
	state  BufferState
	events []bufferEvent
}

func (t *bufferTrace) reset() {
	if t.events != nil {
		t.events = t.events[:0]
	}
}

type BufferMain struct {
	sync.Mutex

	PoolByName map[string]*BufferPool
	bufferPools

	traceByOffset map[uint32]*bufferTrace
}

func (m *BufferMain) AddBufferPool(p *BufferPool) {
	p.m = m
	if len(p.Name) == 0 {
		p.Name = "no-name"
	}
	m.Lock()
	p.ElogName = elog.SetString(p.Name)
	if m.PoolByName == nil {
		m.PoolByName = make(map[string]*BufferPool)
	}
	var exists bool
	if _, exists = m.PoolByName[p.Name]; !exists {
		p.index = uint32(m.bufferPools.GetIndex())
		m.bufferPools.elts[p.index] = p
		m.PoolByName[p.Name] = p
	}
	m.Unlock()
	if !exists {
		p.Init()
	}
}

func (m *BufferMain) DelBufferPool(p *BufferPool) {
	m.Lock()
	delete(m.PoolByName, p.Name)
	m.bufferPools.PutIndex(uint(p.index))
	m.bufferPools.elts[p.index] = nil
	m.Unlock()
	for i := range p.memChunkIDs {
		DmaFree(p.memChunkIDs[i])
	}
	// Unlink garbage.
	p.DmaMemAllocBytes = 0
	p.memChunkIDs = nil
	for i := range p.refs {
		p.unsetState(p.refs[i].offset())
	}
	p.refs = nil
	p.Data = nil
}

func (r *RefHeader) slice(n uint) (l []Ref) {
	var h reflect.SliceHeader
	h.Data = uintptr(unsafe.Pointer(r))
	h.Len = int(n)
	h.Cap = int(n)
	l = *(*[]Ref)(unsafe.Pointer(&h))
	return
}

func (p *BufferPool) FreeLen() uint     { return uint(len(p.refs)) }
func (p *BufferPool) FreeBytes() uint64 { return uint64(p.sizeIncludingOverhead) * uint64(p.FreeLen()) }

func (p *BufferPool) AllocRefs(r *RefHeader, n uint) { p.AllocRefsStride(r, n, 1) }
func (p *BufferPool) AllocRefsStride(r *RefHeader, want, stride uint) {
	if !p.isInitialized() {
		p.Init()
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	got := p.FreeLen()
	total_alloc := uint(0)
	for got < want {
		b := p.sizeIncludingOverhead
		n_alloc := uint(elib.RoundPow2(elib.Word(want-got), 256))
		nb := n_alloc * b
		for nb > 1<<20 {
			n_alloc /= 2
			nb /= 2
		}
		_, id, offset, _ := DmaAlloc(nb)
		ri := got
		p.refs.Resize(n_alloc)
		p.memChunkIDs = append(p.memChunkIDs, id)
		p.DmaMemAllocBytes += uint64(nb)
		// Refs are allocated from end of refs so we put smallest offsets there.
		o := offset + (n_alloc-1)*b
		for i := uint(0); i < n_alloc; i++ {
			// Initialize buffer ref from template.
			r := p.BufferTemplate.Ref
			r.offsetAndFlags += uint32(o)
			p.refs[ri] = r
			ri++
			o -= b

			// Initialize buffer itself from template.
			b := r.GetBuffer()
			*b = p.BufferTemplate.Buffer

			// Initialize buffer data from template.
			if p.BufferTemplate.Data != nil {
				d := r.DataSlice()
				copy(d, p.BufferTemplate.Data)
			}
		}
		got += n_alloc
		total_alloc += n_alloc
		// Possibly initialize/adjust newly made buffers.
		p.InitRefs(p.refs[got-n_alloc : got])

		if elog.Enabled() {
			e := poolElog{
				kind:     pool_elog_grow,
				pool:     p.ElogName,
				n:        uint32(n_alloc),
				nk_alloc: uint32(p.DmaMemAllocBytes / 1024),
				nk_free:  uint32(p.FreeBytes() / 1024),
			}
			elog.Add(&e)
		}
	}

	refs := r.slice(want * stride)
	copyRefs(refs, p.refs[got-want:got], stride)

	p.refs = p.refs[:got-want]
	p.validateSetStateRefs(refs, BufferKnownAllocated, stride)

	if elog.Enabled() {
		e := poolElog{
			kind:     pool_elog_alloc,
			pool:     p.ElogName,
			n:        uint32(want),
			nk_alloc: uint32(p.DmaMemAllocBytes / 1024),
			nk_free:  uint32(p.FreeBytes() / 1024),
		}
		elog.Add(&e)
	}
}

func (p *BufferPool) AllocCachedRefs() (r RefVec) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, p.refs = p.refs, nil
	p.validateSetStateRefs(r, BufferKnownAllocated, 1)
	return
}

func copyRefs(dst, src []Ref, stride uint) {
	if stride == 1 {
		copy(dst, src)
	} else {
		i, ri, n := uint(0), uint(0), uint(len(src))
		for i+4 <= n {
			dst[ri+0*stride] = src[i+0]
			dst[ri+1*stride] = src[i+1]
			dst[ri+2*stride] = src[i+2]
			dst[ri+3*stride] = src[i+3]
			i += 4
			ri += 4 * stride
		}
		for i < n {
			dst[ri+0*stride] = src[i+0]
			i += 1
			ri += 1 * stride
		}
	}
}

type freeNext struct {
	count uint
	refs  RefVec
}

func (f *freeNext) add(p *BufferPool, r *Ref, nextRef RefHeader) {
	if !r.NextIsValid() {
		return
	}
	for {
		p.validateSetState(nextRef, BufferKnownFree)
		f.refs.Validate(f.count)
		f.refs[f.count].RefHeader = nextRef
		f.count++
		if !nextRef.NextIsValid() {
			break
		}
		b := nextRef.GetBuffer()
		nextRef = b.nextRef
	}
}

func (dst *RefHeader) copyOffset(src *Ref) { dst.offsetAndFlags |= src.offset() }

func (p *BufferPool) free4(dst, src []Ref, i uint, tmp *BufferTemplate) (slowPath bool, n0, n1, n2, n3 RefHeader) {
	r0, r1, r2, r3 := &src[i+0], &src[i+1], &src[i+2], &src[i+3]
	t := tmp.Ref
	dst[i+0], dst[i+1], dst[i+2], dst[i+3] = t, t, t, t
	dst[i+0].copyOffset(r0)
	dst[i+1].copyOffset(r1)
	dst[i+2].copyOffset(r2)
	dst[i+3].copyOffset(r3)

	b0, b1, b2, b3 := r0.GetBuffer(), r1.GetBuffer(), r2.GetBuffer(), r3.GetBuffer()
	n0, n1, n2, n3 = b0.nextRef, b1.nextRef, b2.nextRef, b3.nextRef
	save0, save1, save2, save3 := b0.save, b1.save, b2.save, b3.save

	b := tmp.Buffer
	*b0, *b1, *b2, *b3 = b, b, b, b
	b0.save, b1.save, b2.save, b3.save = save0, save1, save2, save3

	slowPath = refFlag4(NextValid, r0, r1, r2, r3)
	return
}

func (p *BufferPool) free1(dst, src []Ref, i uint, tmp *BufferTemplate) (slow bool, n0 RefHeader) {
	r0 := &src[i+0]
	t := tmp.Ref
	dst[i+0] = t
	dst[i+0].copyOffset(r0)

	b0 := r0.GetBuffer()
	n0 = b0.nextRef
	save0 := b0.save

	b := tmp.Buffer
	*b0 = b
	b0.save = save0

	slow = refFlag1(NextValid, r0)
	return
}

func (p *BufferPool) freeRefsNext(dst, src []Ref, n uint, tmp *BufferTemplate) {
	i := uint(0)
	for n >= 4 {
		slow, n0, n1, n2, n3 := p.free4(dst, src, i, tmp)
		i += 4
		n -= 4
		if slow {
			p.freeNext.add(p, &src[i-4], n0)
			p.freeNext.add(p, &src[i-3], n1)
			p.freeNext.add(p, &src[i-2], n2)
			p.freeNext.add(p, &src[i-1], n3)
		}
	}

	for n > 0 {
		slow, n0 := p.free1(dst, src, i, tmp)
		i += 1
		n -= 1
		if slow {
			p.freeNext.add(p, &src[i-1], n0)
		}
	}
}

func (p *BufferPool) freeRefsNoNext(dst, src []Ref, n uint, tmp *BufferTemplate) {
	i := uint(0)
	for n >= 4 {
		p.free4(dst, src, i, tmp)
		i += 4
		n -= 4
	}

	for n > 0 {
		p.free1(dst, src, i, tmp)
		i += 1
		n -= 1
	}
}

// Return all buffers to pool and reset for next usage.
// freeNext specifies whether or not to follow and free next pointers.
func (p *BufferPool) FreeRefs(rh *RefHeader, n uint, freeNext bool) {
	if n == 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	toFree := rh.slice(n)

	p.validateSetStateRefs(toFree, BufferKnownFree, 1)

	initialLen := p.FreeLen()
	p.refs.Resize(n)
	r := p.refs[initialLen:]

	tmp := &p.BufferTemplate
	if !freeNext {
		p.freeRefsNoNext(r, toFree, n, tmp)
	} else {
		fn := &p.freeNext
		fn.count = 0
		p.freeRefsNext(r, toFree, n, tmp)
		if m := fn.count; m > 0 {
			l := len(p.refs)
			p.refs.Resize(m)
			r := p.refs[l:]
			p.freeRefsNoNext(r, fn.refs, m, tmp)
		}
	}

	p.InitRefs(p.refs[initialLen:])

	if elog.Enabled() {
		e := poolElog{
			kind:     pool_elog_free,
			pool:     p.ElogName,
			n:        uint32(n),
			nk_alloc: uint32(p.DmaMemAllocBytes / 1024),
			nk_free:  uint32(p.FreeBytes() / 1024),
		}
		elog.Add(&e)
	}
}

func (r *RefHeader) String() (s string) {
	s = ""
	for {
		if s != "" {
			s += ", "
		}
		s += "{"
		s += fmt.Sprintf("0x%x+%d, %d bytes", r.offset(), r.dataOffset, r.dataLen)
		if f := r.Flags(); f != 0 {
			s += ", " + f.String()
		}
		var ok bool
		if ok = DmaIsValidOffset(uint(r.offset() + uint32(r.dataOffset))); !ok {
			s += ", bad-offset"
		}
		s += "}"
		if !ok {
			break
		}
		if r = r.NextRef(); r == nil {
			break
		}
	}
	return
}

func (h *RefHeader) Validate() {
	if !elib.Debug {
		return
	}
	var err error
	defer func() {
		if err != nil {
			panic(fmt.Errorf("%s %s", h, err))
		}
	}()
	r := h
	for {
		if ok := DmaIsValidOffset(uint(r.offset() + uint32(r.dataOffset))); !ok {
			err = fmt.Errorf("bad dma offset: %x", r.dataOffset)
			return
		}
		if r.DataLen() == 0 {
			err = fmt.Errorf("zero length")
			return
		}
		if r = r.NextRef(); r == nil {
			break
		}
	}
}

// Chains of buffer references.
type RefChain struct {
	// Number of bytes in chain.
	len   uint32
	count uint32
	// Head and tail buffer reference.
	head            Ref
	tail, prev_tail *RefHeader
}

func (c *RefChain) Head() *Ref          { return &c.head }
func (c *RefChain) Len() uint           { return uint(c.len) }
func (c *RefChain) addLen(r *RefHeader) { c.len += uint32(r.dataLen) }

func (c *RefChain) Append(r *RefHeader) {
	c.addLen(r)
	if c.tail == nil {
		c.tail = &c.head.RefHeader
	}
	*c.tail = *r
	if c.prev_tail != nil {
		c.prev_tail.SetFlags(NextValid)
	}
	tail := r
	for {
		// End of chain for reference to be added?
		if x := tail.NextRef(); x == nil {
			c.prev_tail, c.tail = c.tail, &tail.GetBuffer().nextRef
			break
		} else {
			c.addLen(x)
			tail = x
		}
	}
}

// Length in buffer chain.
func (r *RefHeader) ChainLen() (l uint) {
	for {
		l += r.DataLen()
		if r = r.NextRef(); r == nil {
			break
		}
	}
	return
}

func (r *RefHeader) ChainSlice(bʹ []byte) (b []byte) {
	b = bʹ[:0]
	for {
		b = append(b, r.DataSlice()...)
		if r = r.NextRef(); r == nil {
			break
		}
	}
	return
}

func (r *RefHeader) validateTotalLen(want uint) (l uint, ok bool) {
	for {
		l += r.DataLen()
		if r = r.NextRef(); r == nil {
			ok = true
			return
		}
		if l > want {
			ok = false
			return
		}
	}
}

func (c *RefChain) Validate() {
	if !elib.Debug {
		return
	}
	want := c.Len()
	got, ok := c.head.validateTotalLen(want)
	if !ok || got != want {
		panic(fmt.Errorf("length mismatch; got %d != want %d", got, want))
	}
	c.head.Validate()
}

type bufferElog struct {
	pool     elog.StringRef
	offset   uint32
	old, new BufferState
}

func (e *bufferElog) Elog(l *elog.Log) {
	l.Logf("hw buf %v 0x%x %v -> %v", e.pool, e.offset, e.old, e.new)
}

const (
	pool_elog_alloc = iota
	pool_elog_grow
	pool_elog_free
)

type pool_elog_kind uint32

func (k pool_elog_kind) String() string {
	t := [...]string{
		pool_elog_alloc: "alloc",
		pool_elog_grow:  "grow",
		pool_elog_free:  "free",
	}
	return elib.StringerHex(t[:], int(k))
}

type poolElog struct {
	pool     elog.StringRef
	kind     pool_elog_kind
	n        uint32
	nk_alloc uint32
	nk_free  uint32
}

func (e *poolElog) Elog(l *elog.Log) {
	l.Logf("hw buf %v %v %d, %dk alloc, %dk free", e.pool, e.kind, e.n, e.nk_alloc, e.nk_free)
}
