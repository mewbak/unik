// SPDX-License-Identifier: Unlicense OR MIT

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"reflect"
	"time"
	"unsafe"

	_ "eliasnaur.com/unik/kernel"
	virtgpu "eliasnaur.com/unik/virtio/gpu"
	"eliasnaur.com/unik/virtio/input"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/gpu"
	"gioui.org/gpu/backend"
	"gioui.org/gpu/gl"
	"gioui.org/io/pointer"
	"gioui.org/io/router"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type virtBackend struct {
	dev           *virtgpu.Device
	fb            *framebuffer
	texUnits      [maxSamplerUnits]*texture
	samplerViews  [maxSamplerUnits]virtgpu.Handle
	samplerStates [maxSamplerUnits]virtgpu.Handle
	buffers       [1]virtgpu.VertexBuffer
	depth         depthState
	depthCache    map[depthState]virtgpu.Handle
	blend         blendState
	blendCache    map[blendState]virtgpu.Handle
	prog          *program
}

type depthState struct {
	fun    uint32
	enable bool
	mask   bool
}

type blendState struct {
	sfactor, dfactor uint32
	enable           bool
}

type framebuffer struct {
	dev                  *virtgpu.Device
	colorRes, depthRes   virtgpu.Resource
	colorSurf, depthSurf virtgpu.Handle
}

type buffer struct {
	dev    *virtgpu.Device
	res    virtgpu.Resource
	length int
}

type texture struct {
	dev      *virtgpu.Device
	res      virtgpu.Resource
	view     virtgpu.Handle
	state    virtgpu.Handle
	width    int
	height   int
	format   uint32
	released bool
}

type program struct {
	dev           *virtgpu.Device
	minSamplerIdx int
	texUnits      int
	vert          struct {
		shader   virtgpu.Handle
		uniforms *buffer
	}
	frag struct {
		shader   virtgpu.Handle
		uniforms *buffer
	}
}

type inputLayout struct {
	dev         *virtgpu.Device
	vertexElems virtgpu.Handle
	inputs      []backend.InputLocation
	layout      []backend.InputDesc
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

const maxSamplerUnits = 2

func run() error {
	d, err := virtgpu.New()
	if err != nil {
		return err
	}
	var width, height int
	var displayFB *framebuffer
	var colorRes virtgpu.Resource
	var g *gpu.GPU

	inputDev, err := input.New()
	if err != nil {
		return err
	}
	events := make(chan input.Event, 100)
	go func() {
		buf := make([]input.Event, cap(events))
		for {
			n, err := inputDev.Read(buf)
			for i := 0; i < n; i++ {
				events <- buf[i]
			}
			if err != nil {
				panic(err)
			}
		}
	}()
	gofont.Register()
	var queue router.Router
	imap := newInputMapper(inputDev)
	gtx := layout.NewContext(&queue)
	th := material.NewTheme()
	timer := time.NewTimer(0)
	cursor, err := newCursor(d)
	if err != nil {
		return err
	}
	for {
		select {
		case e := <-events:
			imap.event(&queue, e)
		loop:
			for {
				select {
				case e := <-events:
					imap.event(&queue, e)
				default:
					break loop
				}
			}
			d.MoveCursor(cursor, uint32(imap.x+.5), uint32(imap.y+.5))
		case <-d.ConfigNotify():
			if g != nil {
				g.Release()
				displayFB.Release()
				d.CmdCtxDetachResource(colorRes)
				d.CmdResourceUnref(colorRes)
				g = nil
			}
		case <-timer.C:
		}
		if g == nil {
			width, height, err = d.QueryScanout()
			if err != nil {
				return err
			}
			imap.width, imap.height = width, height
			colorRes = createDisplayBuffer(d, width, height)
			fb := newFramebuffer(d, colorRes, virtgpu.VIRGL_FORMAT_B8G8R8A8_SRGB, width, height, 16)
			displayFB = fb
			d.CmdSetScanout(fb.colorRes)
			backend, err := newBackend(d, displayFB)
			if err != nil {
				return err
			}
			backend.BindFramebuffer(displayFB)
			g, err = gpu.New(backend)
			if err != nil {
				return err
			}
		}
		if err := d.Flush3D(); err != nil {
			return err
		}
		sz := image.Point{X: width, Y: height}
		gtx.Reset(&config{1.5}, sz)
		kitchen(gtx, th)
		g.Collect(sz, gtx.Ops)
		g.BeginFrame()
		queue.Frame(gtx.Ops)
		g.EndFrame()
		d.CmdResourceFlush(displayFB.colorRes)
		if t, ok := queue.WakeupTime(); ok {
			timer.Reset(time.Until(t))
		}
	}
}

func newCursor(d *virtgpu.Device) (virtgpu.Resource, error) {
	cursorImg, _, err := image.Decode(bytes.NewBuffer(cursor))
	if err != nil {
		return 0, err
	}
	rgba := image.NewRGBA(cursorImg.Bounds())
	draw.Draw(rgba, rgba.Bounds(), cursorImg, cursorImg.Bounds().Min, draw.Src)
	return d.NewCursor(rgba, image.Point{})
}

type inputMapper struct {
	width, height int
	begun         bool
	xinf          input.AbsInfo
	yinf          input.AbsInfo
	x, y          float32
	buttons       pointer.Buttons
}

func newInputMapper(d *input.Device) *inputMapper {
	m := new(inputMapper)
	m.xinf, _ = d.AbsInfo(input.ABS_X)
	m.yinf, _ = d.AbsInfo(input.ABS_Y)
	return m
}

func (m *inputMapper) event(q *router.Router, e input.Event) {
	switch e.Type {
	case input.EV_SYN:
		if m.begun {
			q.Add(pointer.Event{
				Type:     pointer.Move,
				Source:   pointer.Mouse,
				Position: f32.Point{X: m.x, Y: m.y},
				Buttons:  m.buttons,
			})
			m.begun = false
		}
	case input.EV_REL:
		switch e.Code {
		case input.REL_WHEEL:
			amount := -int32(e.Value)
			q.Add(pointer.Event{
				Type:     pointer.Move,
				Source:   pointer.Mouse,
				Position: f32.Point{X: m.x, Y: m.y},
				Scroll:   f32.Point{Y: float32(amount) * 120},
				Buttons:  m.buttons,
			})
		case input.REL_HWHEEL:
			amount := -int32(e.Value)
			q.Add(pointer.Event{
				Type:     pointer.Move,
				Source:   pointer.Mouse,
				Position: f32.Point{X: m.x, Y: m.y},
				Scroll:   f32.Point{X: float32(amount) * 120},
				Buttons:  m.buttons,
			})
		}
	case input.EV_ABS:
		val := int(e.Value)
		switch e.Code {
		case input.ABS_X:
			m.begun = true
			m.x = m.mapAxis(m.xinf, m.width, val)
		case input.ABS_Y:
			m.begun = true
			m.y = m.mapAxis(m.yinf, m.height, val)
		}
	case input.EV_KEY:
		var button pointer.Buttons
		switch e.Code {
		case input.BTN_LEFT:
			button = pointer.ButtonLeft
		case input.BTN_RIGHT:
			button = pointer.ButtonRight
		case input.BTN_MIDDLE:
			button = pointer.ButtonMiddle
		default:
			return
		}
		var t pointer.Type
		if e.Value != 0 {
			t = pointer.Press
			m.buttons |= button
		} else {
			t = pointer.Release
			m.buttons &^= button
		}
		m.begun = false
		q.Add(pointer.Event{
			Type:     t,
			Source:   pointer.Mouse,
			Position: f32.Point{X: m.x, Y: m.y},
			Buttons:  m.buttons,
		})
	}
}

func (m *inputMapper) mapAxis(inf input.AbsInfo, dim, val int) float32 {
	d := inf.Max - inf.Min
	if d <= 0 {
		return 0
	}
	return float32((val-int(inf.Min))*dim) / float32(d)
}

func createDisplayBuffer(d *virtgpu.Device, width, height int) virtgpu.Resource {
	res := d.CmdResourceCreate3D(virtgpu.ResourceCreate3DReq{
		Format:     virtgpu.VIRGL_FORMAT_B8G8R8A8_SRGB,
		Width:      uint32(width),
		Height:     uint32(height),
		Depth:      1,
		Array_size: 1,
		Flags:      virtgpu.VIRTIO_GPU_RESOURCE_FLAG_Y_0_TOP,
		Bind:       virtgpu.VIRGL_BIND_RENDER_TARGET,
		Target:     virtgpu.PIPE_TEXTURE_2D,
	})
	d.CmdCtxAttachResource(res)
	return res
}

func drawShapes(gtx *layout.Context) {
	blue := color.RGBA{B: 0xFF, A: 0xFF}
	paint.ColorOp{Color: blue}.Add(gtx.Ops)
	paint.PaintOp{Rect: f32.Rectangle{
		Max: f32.Point{X: 50, Y: 100},
	}}.Add(gtx.Ops)

	red := color.RGBA{R: 0xFF, A: 0xFF}
	paint.ColorOp{Color: red}.Add(gtx.Ops)
	paint.PaintOp{Rect: f32.Rectangle{
		Max: f32.Point{X: 100, Y: 50},
	}}.Add(gtx.Ops)

	var stack op.StackOp
	stack.Push(gtx.Ops)
	op.TransformOp{}.Offset(f32.Point{X: 100, Y: 100}).Add(gtx.Ops)
	col := color.RGBA{A: 0xff, R: 0xca, G: 0xfe, B: 0x00}
	col2 := color.RGBA{A: 0xff, R: 0x00, G: 0xfe, B: 0x00}
	pop := paint.PaintOp{Rect: f32.Rectangle{Max: f32.Point{
		X: 500,
		Y: 500,
	}}}
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	clip.Rect{
		Rect: f32.Rectangle{
			Min: f32.Point{X: 50, Y: 50},
			Max: f32.Point{X: 250, Y: 250},
		},
		SE: 15,
	}.Op(gtx.Ops).Add(gtx.Ops)
	pop.Add(gtx.Ops)

	paint.ColorOp{Color: col2}.Add(gtx.Ops)
	clip.Rect{
		Rect: f32.Rectangle{
			Min: f32.Point{X: 100, Y: 100},
			Max: f32.Point{X: 350, Y: 350},
		},
		NW: 25, SE: 115, NE: 35,
	}.Op(gtx.Ops).Add(gtx.Ops)
	pop.Add(gtx.Ops)
	stack.Pop()
}

func newBackend(d *virtgpu.Device, fb *framebuffer) (*virtBackend, error) {
	b := &virtBackend{
		dev:        d,
		fb:         fb,
		blendCache: make(map[blendState]virtgpu.Handle),
		depthCache: make(map[depthState]virtgpu.Handle),
	}
	// Depth mask is on by default.
	b.depth.mask = true
	return b, nil
}

type config struct {
	Scale float32
}

func (s *config) Now() time.Time {
	return time.Now()
}

func (s *config) Px(v unit.Value) int {
	scale := s.Scale
	if v.U == unit.UnitPx {
		scale = 1
	}
	return int(math.Round(float64(scale * v.V)))
}

func (b *virtBackend) BeginFrame() {}

func (b *virtBackend) EndFrame() {
}

func (b *virtBackend) Caps() backend.Caps {
	return backend.Caps{
		MaxTextureSize: 4096, // TODO
	}
}

func (b *virtBackend) NewTimer() backend.Timer {
	panic("timers not implemented")
}

func (b *virtBackend) IsTimeContinuous() bool {
	panic("timers not implemented")
}

func (b *virtBackend) NewFramebuffer(tex backend.Texture, depthBits int) (backend.Framebuffer, error) {
	t := tex.(*texture)
	return newFramebuffer(b.dev, t.res, t.format, t.width, t.height, depthBits), nil
}

func newFramebuffer(d *virtgpu.Device, colorRes virtgpu.Resource, format uint32, width, height, depthBits int) *framebuffer {
	surf := d.CreateSurface(colorRes, format)
	fb := &framebuffer{dev: d, colorRes: colorRes, colorSurf: surf}
	if depthBits > 0 {
		depthRes, depthSurf := createZBuffer(d, width, height)
		fb.depthRes = depthRes
		fb.depthSurf = depthSurf
	}
	return fb
}

func createZBuffer(d *virtgpu.Device, width, height int) (virtgpu.Resource, virtgpu.Handle) {
	res := d.CmdResourceCreate3D(virtgpu.ResourceCreate3DReq{
		Format:     virtgpu.VIRGL_FORMAT_Z24X8_UNORM,
		Width:      uint32(width),
		Height:     uint32(height),
		Depth:      1,
		Array_size: 1,
		Bind:       virtgpu.VIRGL_BIND_DEPTH_STENCIL,
		Target:     virtgpu.PIPE_TEXTURE_2D,
	})
	d.CmdCtxAttachResource(res)
	surf := d.CreateSurface(res, virtgpu.VIRGL_FORMAT_Z24X8_UNORM)
	return res, surf
}

func (b *virtBackend) CurrentFramebuffer() backend.Framebuffer {
	return b.fb
}

func (b *virtBackend) NewTexture(format backend.TextureFormat, width, height int, minFilter, magFilter backend.TextureFilter, binding backend.BufferBinding) (backend.Texture, error) {
	var bfmt uint32
	switch format {
	case backend.TextureFormatSRGB:
		bfmt = virtgpu.VIRGL_FORMAT_B8G8R8A8_SRGB
	case backend.TextureFormatFloat:
		bfmt = virtgpu.VIRGL_FORMAT_R16_FLOAT
	default:
		return nil, fmt.Errorf("gpu: unsupported texture format: %v", format)
	}
	var bind uint32
	if binding&backend.BufferBindingTexture != 0 {
		bind |= virtgpu.VIRGL_BIND_SAMPLER_VIEW
	}
	if binding&backend.BufferBindingFramebuffer != 0 {
		bind |= virtgpu.VIRGL_BIND_RENDER_TARGET
	}
	tex := b.dev.CmdResourceCreate3D(virtgpu.ResourceCreate3DReq{
		Format:     bfmt,
		Width:      uint32(width),
		Height:     uint32(height),
		Depth:      1,
		Array_size: 1,
		//Flags:      virtgpu.VIRTIO_GPU_RESOURCE_FLAG_Y_0_TOP,
		Bind:   bind,
		Target: virtgpu.PIPE_TEXTURE_2D,
	})
	b.dev.CmdCtxAttachResource(tex)
	const swizzle = virtgpu.PIPE_SWIZZLE_ALPHA<<9 | virtgpu.PIPE_SWIZZLE_BLUE<<6 | virtgpu.PIPE_SWIZZLE_GREEN<<3 | virtgpu.PIPE_SWIZZLE_RED
	view := b.dev.CreateSamplerView(tex, bfmt, virtgpu.PIPE_TEXTURE_2D, swizzle)
	minImg := convertTextureFilter(minFilter)
	magImg := convertTextureFilter(magFilter)

	state := b.dev.CreateSamplerState(
		virtgpu.PIPE_TEX_WRAP_CLAMP_TO_EDGE,
		virtgpu.PIPE_TEX_WRAP_CLAMP_TO_EDGE,
		minImg, virtgpu.PIPE_TEX_MIPFILTER_NONE,
		magImg,
	)
	return &texture{dev: b.dev, res: tex, view: view, state: state, width: width, height: height, format: bfmt}, nil
}

func convertTextureFilter(f backend.TextureFilter) uint32 {
	switch f {
	case backend.FilterLinear:
		return virtgpu.PIPE_TEX_FILTER_LINEAR
	case backend.FilterNearest:
		return virtgpu.PIPE_TEX_FILTER_NEAREST
	default:
		panic("unknown texture filter")
	}
}

func (b *virtBackend) NewBuffer(typ backend.BufferBinding, size int) (backend.Buffer, error) {
	return b.newBuffer(typ, size)
}

func (b *virtBackend) newBuffer(typ backend.BufferBinding, length int) (*buffer, error) {
	bind, err := convBufferBinding(typ)
	if err != nil {
		return nil, err
	}
	res := b.dev.CmdResourceCreate3D(virtgpu.ResourceCreate3DReq{
		Width:      uint32(length),
		Height:     1,
		Depth:      1,
		Array_size: 1,
		Bind:       bind,
		Target:     virtgpu.PIPE_BUFFER,
	})
	b.dev.CmdCtxAttachResource(res)
	return &buffer{dev: b.dev, res: res, length: length}, nil
}

func (b *virtBackend) NewImmutableBuffer(typ backend.BufferBinding, data []byte) (backend.Buffer, error) {
	buf, err := b.newBuffer(typ, len(data))
	if err != nil {
		return nil, err
	}
	return buf, buf.upload(data)
}

func (b *virtBackend) NewInputLayout(vs backend.ShaderSources, layout []backend.InputDesc) (backend.InputLayout, error) {
	elems := make([]virtgpu.VertexElement, len(vs.Inputs))
	for i, input := range vs.Inputs {
		vfmt, err := convertVertexFormat(input.Type, input.Size)
		if err != nil {
			return nil, err
		}
		l := layout[i]
		elems[i] = virtgpu.VertexElement{
			Format:      vfmt,
			Offset:      uint32(l.Offset),
			Divisor:     0,
			BufferIndex: 0, // Only one vertex buffer is supported.
		}
	}
	ve, err := b.dev.CreateVertexElements(elems)
	if err != nil {
		return nil, err
	}
	return &inputLayout{
		dev:         b.dev,
		vertexElems: ve,
		inputs:      vs.Inputs,
		layout:      layout,
	}, nil
}

func convertVertexFormat(dataType backend.DataType, size int) (uint32, error) {
	var f uint32
	switch dataType {
	case backend.DataTypeFloat:
		switch size {
		case 1:
			f = virtgpu.VIRGL_FORMAT_R32_FLOAT
		case 2:
			f = virtgpu.VIRGL_FORMAT_R32G32_FLOAT
		case 3:
			f = virtgpu.VIRGL_FORMAT_R32G32B32_FLOAT
		case 4:
			f = virtgpu.VIRGL_FORMAT_R32G32B32A32_FLOAT
		}
	default:
		return 0, fmt.Errorf("gpu: invalid data type %v, size %d", dataType, size)
	}
	return f, nil
}

func (b *virtBackend) NewProgram(vertShader, fragShader backend.ShaderSources) (backend.Program, error) {
	tgsi, exist := shaders[[2]string{fragShader.GLSL100ES, vertShader.GLSL100ES}]
	if !exist {
		return nil, fmt.Errorf("gpu: unrecognized vertex shader")
	}
	fsrc, vsrc := tgsi[0], tgsi[1]
	vh := b.dev.CreateShader(virtgpu.PIPE_SHADER_VERTEX, vsrc)
	fh := b.dev.CreateShader(virtgpu.PIPE_SHADER_FRAGMENT, fsrc)
	minSamplerIdx := maxSamplerUnits
	for _, t := range fragShader.Textures {
		if t.Binding < minSamplerIdx {
			minSamplerIdx = t.Binding
		}
	}
	p := &program{dev: b.dev, minSamplerIdx: minSamplerIdx, texUnits: len(fragShader.Textures)}
	p.vert.shader = vh
	p.frag.shader = fh
	return p, nil
}

func (b *virtBackend) SetDepthTest(enable bool) {
	b.depth.enable = enable
}

func (b *virtBackend) DepthMask(mask bool) {
	b.depth.mask = mask
}

func (b *virtBackend) DepthFunc(fun backend.DepthFunc) {
	var f uint32
	switch fun {
	case backend.DepthFuncGreater:
		f = virtgpu.DepthFuncGreater
	default:
		panic("unsupported depth func")
	}
	b.depth.fun = f
}

func (b *virtBackend) BlendFunc(sfactor, dfactor backend.BlendFactor) {
	b.blend.sfactor = convertBlendFactor(sfactor)
	b.blend.dfactor = convertBlendFactor(dfactor)
}

func convertBlendFactor(f backend.BlendFactor) uint32 {
	switch f {
	case backend.BlendFactorOne:
		return virtgpu.PIPE_BLENDFACTOR_ONE
	case backend.BlendFactorOneMinusSrcAlpha:
		return virtgpu.PIPE_BLENDFACTOR_INV_SRC_ALPHA
	case backend.BlendFactorZero:
		return virtgpu.PIPE_BLENDFACTOR_ZERO
	case backend.BlendFactorDstColor:
		return virtgpu.PIPE_BLENDFACTOR_DST_COLOR
	default:
		panic("unsupported blend factor")
	}
}

func (b *virtBackend) SetBlend(enable bool) {
	b.blend.enable = enable
}

func (b *virtBackend) DrawElements(mode backend.DrawMode, off, count int) {
	b.prepareDraw()
	m := convertDrawMode(mode)
	b.dev.Draw(true, m, uint32(off), uint32(count))
}

func (b *virtBackend) DrawArrays(mode backend.DrawMode, off, count int) {
	b.prepareDraw()
	m := convertDrawMode(mode)
	b.dev.Draw(false, m, uint32(off), uint32(count))
}

func convertDrawMode(mode backend.DrawMode) uint32 {
	switch mode {
	case backend.DrawModeTriangles:
		return gl.TRIANGLES
	case backend.DrawModeTriangleStrip:
		return gl.TRIANGLE_STRIP
	default:
		panic("unsupported draw mode")
	}
}

func (b *virtBackend) prepareDraw() {
	if p := b.prog; p != nil {
		if u := p.vert.uniforms; u != nil {
			b.dev.SetUniformBuffer(virtgpu.PIPE_SHADER_VERTEX, 1, 0, uint32(u.length), u.res)
		}
		if u := p.frag.uniforms; u != nil {
			b.dev.SetUniformBuffer(virtgpu.PIPE_SHADER_FRAGMENT, 1, 0, uint32(u.length), u.res)
		}
		for i := 0; i < p.texUnits; i++ {
			u := i + p.minSamplerIdx
			if t := b.texUnits[u]; t != nil && !t.released {
				b.samplerStates[i] = t.state
				b.samplerViews[i] = t.view
			} else {
				b.samplerStates[i] = 0
				b.samplerViews[i] = 0
			}
		}
		b.dev.SetSamplerStates(virtgpu.PIPE_SHADER_FRAGMENT, b.samplerStates[:p.texUnits])
		b.dev.SetSamplerViews(virtgpu.PIPE_SHADER_FRAGMENT, b.samplerViews[:p.texUnits])
	}
	bstate, exists := b.blendCache[b.blend]
	if !exists {
		bstate = b.dev.CreateBlend(b.blend.enable, virtgpu.PIPE_BLEND_ADD, b.blend.sfactor, b.blend.dfactor)
		b.blendCache[b.blend] = bstate
	}
	b.dev.BindObject(virtgpu.VIRGL_OBJECT_BLEND, bstate)
	dstate, exists := b.depthCache[b.depth]
	if !exists {
		dstate = b.dev.CreateDepthState(b.depth.enable, b.depth.mask, b.depth.fun)
		b.depthCache[b.depth] = dstate
	}
	b.dev.BindObject(virtgpu.VIRGL_OBJECT_DSA, dstate)
}

func (b *virtBackend) Viewport(x, y, width, height int) {
	b.dev.Viewport(x, y, width, height)
}

func (b *virtBackend) ClearDepth(d float32) {
	b.dev.Clear(virtgpu.PIPE_CLEAR_DEPTH, [4]float32{}, d)
}

func (b *virtBackend) Clear(colR, colG, colB, colA float32) {
	b.dev.Clear(virtgpu.PIPE_CLEAR_COLOR0, [4]float32{colR, colB, colG, colA}, 0)
}

func (b *virtBackend) BindProgram(prog backend.Program) {
	p := prog.(*program)
	b.dev.BindShader(virtgpu.PIPE_SHADER_VERTEX, p.vert.shader)
	b.dev.BindShader(virtgpu.PIPE_SHADER_FRAGMENT, p.frag.shader)
	b.prog = p
}

func (b *virtBackend) BindVertexBuffer(buf backend.Buffer, stride, offset int) {
	res := buf.(*buffer).res
	b.buffers[0].Stride = uint32(stride)
	b.buffers[0].Offset = uint32(offset)
	b.buffers[0].Buffer = res
	b.dev.SetVertexBuffers(b.buffers[:])
}

func (b *virtBackend) BindIndexBuffer(buf backend.Buffer) {
	const uint16Size = 2
	b.dev.SetIndexBuffer(buf.(*buffer).res, uint16Size, 0)
}

func (b *virtBackend) BindFramebuffer(fbo backend.Framebuffer) {
	f := fbo.(*framebuffer)
	b.dev.SetFramebufferState(f.colorSurf, f.depthSurf)
}

func (b *virtBackend) BindTexture(unit int, tex backend.Texture) {
	t := tex.(*texture)
	b.texUnits[unit] = t
}

func (b *virtBackend) BindInputLayout(layout backend.InputLayout) {
	l := layout.(*inputLayout)
	b.dev.BindObject(virtgpu.VIRGL_OBJECT_VERTEX_ELEMENTS, l.vertexElems)
}

func (b *virtBackend) Release() {
	for _, state := range b.blendCache {
		b.dev.DestroyObject(state)
	}
	for _, state := range b.depthCache {
		b.dev.DestroyObject(state)
	}
}

func (f *framebuffer) Invalidate() {}

func (f *framebuffer) ReadPixels(rect image.Rectangle, pix []byte) error {
	panic("TODO")
}

func (f *framebuffer) Release() {
	if f.colorSurf != 0 {
		f.dev.DestroyObject(f.colorSurf)
	}
	if f.depthSurf != 0 {
		f.dev.DestroyObject(f.depthSurf)
	}
	if f.depthRes != 0 {
		f.dev.CmdCtxDetachResource(f.depthRes)
		f.dev.CmdResourceUnref(f.depthRes)
	}
}

func (b *buffer) Release() {
	b.dev.CmdCtxDetachResource(b.res)
	b.dev.CmdResourceUnref(b.res)
}

func (t *texture) Release() {
	t.dev.DestroyObject(t.state)
	t.dev.DestroyObject(t.view)
	t.dev.CmdCtxDetachResource(t.res)
	t.dev.CmdResourceUnref(t.res)
	t.released = true
}

func (p *program) SetVertexUniforms(uniforms backend.Buffer) {
	p.vert.uniforms = uniforms.(*buffer)
}

func (p *program) SetFragmentUniforms(uniforms backend.Buffer) {
	p.frag.uniforms = uniforms.(*buffer)
}

func (p *program) Release() {
	p.dev.DestroyObject(p.frag.shader)
	p.dev.DestroyObject(p.vert.shader)
}

func (b *buffer) upload(data []byte) error {
	b.dev.Copy(b.res, data, len(data), 1)
	return nil
}

func (b *buffer) Upload(data []byte) {
	if err := b.upload(data); err != nil {
		panic(err)
	}
}

func (t *texture) upload(data []byte, width, height int) error {
	t.dev.Copy(t.res, data, width, height)
	return nil
}

func (t *texture) Upload(img *image.RGBA) {
	var pixels []byte
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if img.Stride != w*4 {
		panic("unsupported stride")
	}
	start := (b.Min.X + b.Min.Y*w) * 4
	end := (b.Max.X + (b.Max.Y-1)*w) * 4
	pixels = img.Pix[start:end]
	if err := t.upload(pixels, w, h); err != nil {
		panic(err)
	}
}

func (i *inputLayout) Release() {
	i.dev.DestroyObject(i.vertexElems)
}

func convBufferBinding(typ backend.BufferBinding) (uint32, error) {
	var res uint32
	switch typ {
	case backend.BufferBindingIndices:
		res = virtgpu.VIRGL_BIND_INDEX_BUFFER
	case backend.BufferBindingVertices:
		res = virtgpu.VIRGL_BIND_VERTEX_BUFFER
	case backend.BufferBindingUniforms:
		res = virtgpu.VIRGL_BIND_CONSTANT_BUFFER
	default:
		return 0, fmt.Errorf("gpu: unsupported BufferBinding: %v", typ)
	}
	return res, nil
}

func testSimpleShader(b backend.Device) error {
	p, err := b.NewProgram(shader_simple_vert, shader_simple_frag)
	if err != nil {
		return err
	}
	defer p.Release()
	b.BindProgram(p)
	layout, err := b.NewInputLayout(shader_simple_vert, nil)
	if err != nil {
		return err
	}
	defer layout.Release()
	b.BindInputLayout(layout)
	b.DrawArrays(backend.DrawModeTriangles, 0, 3)
	return nil
}

func testInputShader(b backend.Device) error {
	p, err := b.NewProgram(shader_input_vert, shader_simple_frag)
	if err != nil {
		return err
	}
	defer p.Release()
	b.BindProgram(p)
	buf, err := b.NewImmutableBuffer(backend.BufferBindingVertices,
		BytesView([]float32{
			0, .5, .5, 1,
			-.5, -.5, .5, 1,
			.5, -.5, .5, 1,
		}),
	)
	if err != nil {
		return err
	}
	defer buf.Release()
	b.BindVertexBuffer(buf, 4*4, 0)
	layout, err := b.NewInputLayout(shader_input_vert, []backend.InputDesc{
		{
			Type:   backend.DataTypeFloat,
			Size:   4,
			Offset: 0,
		},
	})
	if err != nil {
		return err
	}
	defer layout.Release()
	b.BindInputLayout(layout)
	b.DrawArrays(backend.DrawModeTriangles, 0, 3)
	return nil
}

// BytesView returns a byte slice view of a slice.
func BytesView(s interface{}) []byte {
	v := reflect.ValueOf(s)
	first := v.Index(0)
	sz := int(first.Type().Size())
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(first.UnsafeAddr())))),
		Len:  v.Len() * sz,
		Cap:  v.Cap() * sz,
	}))
}

func setupFBO(b backend.Device, size image.Point) (backend.Texture, backend.Framebuffer, error) {
	tex, fbo, err := newFBO(b, size)
	if err != nil {
		return nil, nil, err
	}
	b.BindFramebuffer(fbo)
	// ClearColor accepts linear RGBA colors, while 8-bit colors
	// are in the sRGB color space.
	var clearCol = color.RGBA{A: 0xff, R: 0xde, G: 0xad, B: 0xbe}
	col := RGBAFromSRGB(clearCol)
	b.Clear(col.Float32())
	b.ClearDepth(0.0)
	b.Viewport(0, 0, size.X, size.Y)
	return tex, fbo, nil
}

func newFBO(b backend.Device, size image.Point) (backend.Texture, backend.Framebuffer, error) {
	fboTex, err := b.NewTexture(
		backend.TextureFormatSRGB,
		size.X, size.Y,
		backend.FilterNearest, backend.FilterNearest,
		backend.BufferBindingFramebuffer,
	)
	if err != nil {
		return nil, nil, err
	}
	const depthBits = 16
	fbo, err := b.NewFramebuffer(fboTex, depthBits)
	if err != nil {
		fboTex.Release()
		return nil, nil, err
	}
	return fboTex, fbo, nil
}

// RGBAFromSRGB converts color.Color to RGBA.
func RGBAFromSRGB(col color.Color) RGBA {
	r, g, b, a := col.RGBA()
	return RGBA{
		R: sRGBToLinear(float32(r) / 0xffff),
		G: sRGBToLinear(float32(g) / 0xffff),
		B: sRGBToLinear(float32(b) / 0xffff),
		A: float32(a) / 0xFFFF,
	}
}

// RGBA is a 32 bit floating point linear space color.
type RGBA struct {
	R, G, B, A float32
}

// sRGBToLinear transforms color value from sRGB to linear.
func sRGBToLinear(c float32) float32 {
	// Formula from EXT_sRGB.
	if c <= 0.04045 {
		return c / 12.92
	} else {
		return float32(math.Pow(float64((c+0.055)/1.055), 2.4))
	}
}

// Float32 returns r, g, b, a values.
func (col RGBA) Float32() (r, g, b, a float32) {
	return col.R, col.G, col.B, col.A
}
