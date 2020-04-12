// SPDX-License-Identifier: Unlicense OR MIT

package gpu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"math"
	"unsafe"

	_ "image/png"

	"eliasnaur.com/unik/virtio"
)

type Device struct {
	dev *virtio.Device
	cfg struct {
		cfg    *config
		notify <-chan struct{}
	}
	cmd struct {
		q *virtio.Commander

		buf    *virtio.IOMem
		bufOff int
	}
	cmd3d struct {
		begun  bool
		offset int
		size   int
	}
	cursor struct {
		q   *virtio.Commander
		buf *virtio.IOMem
	}

	scanout struct {
		id   uint32
		rect rect
	}

	ctxID uint32

	nextID uint32

	submitBuf bytes.Buffer
	submitErr error
}

type SamplerView struct {
	Slot int
	View Handle
}

type Resource uint32

type Handle uint32

type VertexBuffer struct {
	Stride uint32
	Offset uint32
	Buffer Resource
}

type VertexElement struct {
	Offset      uint32
	Divisor     uint32
	BufferIndex uint32
	Format      uint32
}

type config struct {
	events_read  uint32
	events_clear uint32
	num_scanouts uint32
	reserved     uint32
}

type ctrlHdr struct {
	_type    uint32
	flags    uint32
	fence_id uint64
	ctx_id   uint32
	padding  uint32
}

type rect struct {
	x      uint32
	y      uint32
	width  uint32
	height uint32
}

type getCapsetInfoReq struct {
	hdr          ctrlHdr
	capset_index uint32
	padding      uint32
}

type capsetInfoResp struct {
	hdr                ctrlHdr
	capset_id          uint32
	capset_max_version uint32
	capset_max_size    uint32
	padding            uint32
}

type getCapsetReq struct {
	hdr            ctrlHdr
	capset_id      uint32
	capset_version uint32
}

type displayInfoResp struct {
	hdr    ctrlHdr
	pmodes [_VIRTIO_GPU_MAX_SCANOUTS]struct {
		r       rect
		enabled uint32
		flags   uint32
	}
}

type resourceCreate2DReq struct {
	hdr         ctrlHdr
	resource_id Resource
	format      uint32
	width       uint32
	height      uint32
}

type setScanoutReq struct {
	hdr         ctrlHdr
	r           rect
	scanout_id  uint32
	resource_id Resource
}

type resourceAttachBackingReq struct {
	hdr         ctrlHdr
	resource_id Resource
	nr_entries  uint32
}

type resourceDetachBackingReq struct {
	hdr         ctrlHdr
	resource_id Resource
	padding     uint32
}

type transferToHost2DReq struct {
	hdr         ctrlHdr
	r           rect
	offset      uint64
	resource_id Resource
	padding     uint32
}

type resourceFlushReq struct {
	hdr         ctrlHdr
	r           rect
	resource_id Resource
	padding     uint32
}

type memEntry struct {
	addr    uint64
	length  uint32
	padding uint32
}

type ctxCreateReq struct {
	hdr        ctrlHdr
	nlen       uint32
	padding    uint32
	debug_name [64]byte
}

type ctxResourceReq struct {
	hdr         ctrlHdr
	resource_id Resource
	padding     uint32
}

type cmdSubmitReq struct {
	hdr     ctrlHdr
	size    uint32
	padding uint32
}

type resourceUnrefReq struct {
	hdr         ctrlHdr
	resource_id Resource
	padding     uint32
}

type ResourceCreate3DReq struct {
	hdr         ctrlHdr
	resource_id Resource
	Target      uint32
	Format      uint32
	Bind        uint32
	Width       uint32
	Height      uint32
	Depth       uint32
	Array_size  uint32
	last_level  uint32
	nr_samples  uint32
	Flags       uint32
	padding     uint32
}

type cursorPos struct {
	scanout_id uint32
	x          uint32
	y          uint32
	padding    uint32
}

type updateCursorReq struct {
	hdr         ctrlHdr
	pos         cursorPos
	resource_id Resource
	hot_x       uint32
	hot_y       uint32
	padding     uint32
}

type supportedFormatMask struct {
	bitmask [16]uint32
}

type capsV1 struct {
	max_version                    uint32
	sampler                        supportedFormatMask
	render                         supportedFormatMask
	depthstencil                   supportedFormatMask
	vertexbuffer                   supportedFormatMask
	bset                           uint32
	glsl_level                     uint32
	max_texture_array_layers       uint32
	max_streamout_buffers          uint32
	max_dual_source_render_targets uint32
	max_render_targets             uint32
	max_samples                    uint32
	prim_mask                      uint32
	max_tbo_size                   uint32
	max_uniform_blocks             uint32
	max_viewports                  uint32
	max_texture_gather_components  uint32
}

type capsV2 struct {
	v1                                  capsV1
	min_aliased_point_size              float32
	max_aliased_point_size              float32
	min_smooth_point_size               float32
	max_smooth_point_size               float32
	min_aliased_line_width              float32
	max_aliased_line_width              float32
	min_smooth_line_width               float32
	max_smooth_line_width               float32
	max_texture_lod_bias                float32
	max_geom_output_vertices            uint32
	max_geom_total_output_components    uint32
	max_vertex_outputs                  uint32
	max_vertex_attribs                  uint32
	max_shader_patch_varyings           uint32
	min_texel_offset                    int32
	max_texel_offset                    int32
	min_texture_gather_offset           int32
	max_texture_gather_offset           int32
	texture_buffer_offset_alignment     uint32
	uniform_buffer_offset_alignment     uint32
	shader_buffer_offset_alignment      uint32
	capability_bits                     uint32
	sample_locations                    [8]uint32
	max_vertex_attrib_stride            uint32
	max_shader_buffer_frag_compute      uint32
	max_shader_buffer_other_stages      uint32
	max_shader_image_frag_compute       uint32
	max_shader_image_other_stages       uint32
	max_image_samples                   uint32
	max_compute_work_group_invocations  uint32
	max_compute_shared_memory_size      uint32
	max_compute_grid_size               [3]uint32
	max_compute_block_size              [3]uint32
	max_texture_2d_size                 uint32
	max_texture_3d_size                 uint32
	max_texture_cube_size               uint32
	max_combined_shader_buffers         uint32
	max_atomic_counters                 [6]uint32
	max_atomic_counter_buffers          [6]uint32
	max_combined_atomic_counters        uint32
	max_combined_atomic_counter_buffers uint32
	host_feature_check_version          uint32
	supported_readback_formats          supportedFormatMask
	scanout                             supportedFormatMask
}

const (
	// 3D support.
	_VIRTIO_GPU_F_VIRGL = 1 << 0

	_VIRTIO_GPU_MAX_SCANOUTS = 16

	virtGPUControlQueue = 0
	virtGPUCursorQueue  = 1
)

const (
	VIRTIO_GPU_RESOURCE_FLAG_Y_0_TOP = 1 << 0
)

const (
	_VIRTIO_GPU_CMD_GET_DISPLAY_INFO = 0x0100 + iota
	_VIRTIO_GPU_CMD_RESOURCE_CREATE_2D
	_VIRTIO_GPU_CMD_RESOURCE_UNREF
	_VIRTIO_GPU_CMD_SET_SCANOUT
	_VIRTIO_GPU_CMD_RESOURCE_FLUSH
	_VIRTIO_GPU_CMD_TRANSFER_TO_HOST_2D
	_VIRTIO_GPU_CMD_RESOURCE_ATTACH_BACKING
	_VIRTIO_GPU_CMD_RESOURCE_DETACH_BACKING
	_VIRTIO_GPU_CMD_GET_CAPSET_INFO
	_VIRTIO_GPU_CMD_GET_CAPSET
	_VIRTIO_GPU_CMD_GET_EDID
)

const (
	_VIRTIO_GPU_CMD_CTX_CREATE = 0x0200 + iota
	_VIRTIO_GPU_CMD_CTX_DESTROY
	_VIRTIO_GPU_CMD_CTX_ATTACH_RESOURCE
	_VIRTIO_GPU_CMD_CTX_DETACH_RESOURCE
	_VIRTIO_GPU_CMD_RESOURCE_CREATE_3D
	_VIRTIO_GPU_CMD_TRANSFER_TO_HOST_3D
	_VIRTIO_GPU_CMD_TRANSFER_FROM_HOST_3D
	_VIRTIO_GPU_CMD_SUBMIT_3D
)

const (
	_VIRTIO_GPU_CMD_UPDATE_CURSOR = 0x0300 + iota
	_VIRTIO_GPU_CMD_MOVE_CURSOR
)

const (
	_VIRTIO_GPU_RESP_OK_NODATA = 0x1100 + iota
	_VIRTIO_GPU_RESP_OK_DISPLAY_INFO
	_VIRTIO_GPU_RESP_OK_CAPSET_INFO
	_VIRTIO_GPU_RESP_OK_CAPSET
	_VIRTIO_GPU_RESP_OK_EDID
)

const (
	_VIRTIO_GPU_RESP_ERR_UNSPEC = 0x1200 + iota
	_VIRTIO_GPU_RESP_ERR_OUT_OF_MEMORY
	_VIRTIO_GPU_RESP_ERR_INVALID_SCANOUT_ID
	_VIRTIO_GPU_RESP_ERR_INVALID_RESOURCE_ID
	_VIRTIO_GPU_RESP_ERR_INVALID_CONTEXT_ID
	_VIRTIO_GPU_RESP_ERR_INVALID_PARAMETER
)

const (
	_VIRGL_FORMAT_NONE                = 0
	_VIRTIO_GPU_FORMAT_B8G8R8A8_UNORM = 1
	_VIRTIO_GPU_FORMAT_B8G8R8X8_UNORM = 2
	_VIRTIO_GPU_FORMAT_A8R8G8B8_UNORM = 3
	_VIRTIO_GPU_FORMAT_X8R8G8B8_UNORM = 4

	_VIRTIO_GPU_FORMAT_R8G8B8A8_UNORM = 67
	_VIRTIO_GPU_FORMAT_X8B8G8R8_UNORM = 68

	_VIRTIO_GPU_FORMAT_A8B8G8R8_UNORM = 121
	_VIRTIO_GPU_FORMAT_R8G8B8X8_UNORM = 134

	VIRGL_FORMAT_B8G8R8A8_SRGB = 100
	VIRGL_FORMAT_A8R8G8B8_SRGB = 102
	VIRGL_FORMAT_Z24X8_UNORM   = 21
	VIRGL_FORMAT_R16_FLOAT     = 91

	VIRGL_FORMAT_R32_FLOAT          = 28
	VIRGL_FORMAT_R32G32_FLOAT       = 29
	VIRGL_FORMAT_R32G32B32_FLOAT    = 30
	VIRGL_FORMAT_R32G32B32A32_FLOAT = 31
)

const (
	PIPE_TEX_MIPFILTER_NONE = 2
)

const (
	PIPE_TEX_FILTER_NEAREST = 0
	PIPE_TEX_FILTER_LINEAR  = 1
)

const (
	_VIRTIO_GPU_FLAG_FENCE = 1 << 0
)

const (
	_VIRGL_CCMD_NOP = iota
	_VIRGL_CCMD_CREATE_OBJECT
	_VIRGL_CCMD_BIND_OBJECT
	_VIRGL_CCMD_DESTROY_OBJECT
	_VIRGL_CCMD_SET_VIEWPORT_STATE
	_VIRGL_CCMD_SET_FRAMEBUFFER_STATE
	_VIRGL_CCMD_SET_VERTEX_BUFFERS
	_VIRGL_CCMD_CLEAR
	_VIRGL_CCMD_DRAW_VBO
	_VIRGL_CCMD_RESOURCE_INLINE_WRITE
	_VIRGL_CCMD_SET_SAMPLER_VIEWS
	_VIRGL_CCMD_SET_INDEX_BUFFER
	_VIRGL_CCMD_SET_CONSTANT_BUFFER
	_VIRGL_CCMD_SET_STENCIL_REF
	_VIRGL_CCMD_SET_BLEND_COLOR
	_VIRGL_CCMD_SET_SCISSOR_STATE
	_VIRGL_CCMD_BLIT
	_VIRGL_CCMD_RESOURCE_COPY_REGION
	_VIRGL_CCMD_BIND_SAMPLER_STATES
	_VIRGL_CCMD_BEGIN_QUERY
	_VIRGL_CCMD_END_QUERY
	_VIRGL_CCMD_GET_QUERY_RESULT
	_VIRGL_CCMD_SET_POLYGON_STIPPLE
	_VIRGL_CCMD_SET_CLIP_STATE
	_VIRGL_CCMD_SET_SAMPLE_MASK
	_VIRGL_CCMD_SET_STREAMOUT_TARGETS
	_VIRGL_CCMD_SET_RENDER_CONDITION
	_VIRGL_CCMD_SET_UNIFORM_BUFFER

	_VIRGL_CCMD_SET_SUB_CTX
	_VIRGL_CCMD_CREATE_SUB_CTX
	_VIRGL_CCMD_DESTROY_SUB_CTX
	_VIRGL_CCMD_BIND_SHADER
	_VIRGL_CCMD_SET_TESS_STATE
	_VIRGL_CCMD_SET_MIN_SAMPLES
	_VIRGL_CCMD_SET_SHADER_BUFFERS
	_VIRGL_CCMD_SET_SHADER_IMAGES
	_VIRGL_CCMD_MEMORY_BARRIER
	_VIRGL_CCMD_LAUNCH_GRID
	_VIRGL_CCMD_SET_FRAMEBUFFER_STATE_NO_ATTACH
	_VIRGL_CCMD_TEXTURE_BARRIER
	_VIRGL_CCMD_SET_ATOMIC_BUFFERS
	_VIRGL_CCMD_SET_DEBUG_FLAGS
	_VIRGL_CCMD_GET_QUERY_RESULT_QBO
	_VIRGL_CCMD_TRANSFER3D
	_VIRGL_CCMD_END_TRANSFERS
	_VIRGL_CCMD_COPY_TRANSFER3D
	_VIRGL_CCMD_SET_TWEAKS
	_VIRGL_MAX_COMMANDS
)

const (
	PIPE_BLEND_ADD = 0
)

const (
	PIPE_TEX_WRAP_CLAMP_TO_EDGE = 2
)

const (
	VIRGL_BIND_DEPTH_STENCIL   = 1 << 0
	VIRGL_BIND_RENDER_TARGET   = 1 << 1
	VIRGL_BIND_SAMPLER_VIEW    = 1 << 3
	VIRGL_BIND_VERTEX_BUFFER   = 1 << 4
	VIRGL_BIND_INDEX_BUFFER    = 1 << 5
	VIRGL_BIND_CONSTANT_BUFFER = 1 << 6
	_VIRGL_BIND_STAGING        = 1 << 19
)

const (
	VIRGL_OBJECT_BLEND           = 1
	VIRGL_OBJECT_DSA             = 3
	_VIRGL_OBJECT_SHADER         = 4
	VIRGL_OBJECT_VERTEX_ELEMENTS = 5
	_VIRGL_OBJECT_SAMPLER_VIEW   = 6
	_VIRGL_OBJECT_SAMPLER_STATE  = 7
	_VIRGL_OBJECT_SURFACE        = 8
)

const (
	DepthFuncGreater = 4
)

const (
	PIPE_BUFFER     = 0
	PIPE_TEXTURE_2D = 2
)

const (
	PIPE_SWIZZLE_RED   = 0
	PIPE_SWIZZLE_GREEN = 1
	PIPE_SWIZZLE_BLUE  = 2
	PIPE_SWIZZLE_ALPHA = 3
)

const (
	PIPE_BLENDFACTOR_ONE           = 0x1
	PIPE_BLENDFACTOR_ZERO          = 0x11
	PIPE_BLENDFACTOR_INV_SRC_ALPHA = 0x13
	PIPE_BLENDFACTOR_DST_COLOR     = 0x5
)

const (
	PIPE_CLEAR_DEPTH   = 1 << 0
	PIPE_CLEAR_STENCIL = 1 << 1
	PIPE_CLEAR_COLOR0  = 1 << 2
	_PIPE_CLEAR_COLOR1 = 1 << 3
	_PIPE_CLEAR_COLOR2 = 1 << 4
	_PIPE_CLEAR_COLOR3 = 1 << 5
	_PIPE_CLEAR_COLOR4 = 1 << 6
	_PIPE_CLEAR_COLOR5 = 1 << 7
	_PIPE_CLEAR_COLOR6 = 1 << 8
	_PIPE_CLEAR_COLOR7 = 1 << 9
)

const (
	PIPE_SHADER_VERTEX   = 0
	PIPE_SHADER_FRAGMENT = 1
)

const PIPE_MASK_RGBA = 0xf

const (
	_VIRTIO_GPU_EVENT_DISPLAY = 1 << 0
)

const (
	_VIRGL_CAP_COPY_TRANSFER = 1 << 26
)

func New() (*Device, error) {
	const deviceTypeGPU = 16
	vdev, err := virtio.New(deviceTypeGPU)
	if err != nil {
		return nil, err
	}
	d, err := newDevice(vdev)
	if err != nil {
		return nil, err
	}
	caps, err := d.queryCaps()
	if err != nil {
		return nil, err
	}
	if caps.capability_bits&_VIRGL_CAP_COPY_TRANSFER == 0 {
		return nil, errors.New("virtgpu: VIRGL_CAP_COPY_TRANSFER not supported")
	}
	ctxID := d.cmdCtxCreate()
	if err := d.Flush3D(); err != nil {
		return nil, err
	}
	d.ctxID = ctxID
	return d, nil
}

func newDevice(dev *virtio.Device) (*Device, error) {
	devCfgMap, _, err := dev.FindCapability(virtio.PCI_CAP_DEVICE_CFG)
	if err != nil {
		return nil, err
	}
	if unsafe.Sizeof(config{}) > uintptr(len(devCfgMap)) {
		return nil, errors.New("gpu: device configuration area too small")
	}
	cfg := (*config)(unsafe.Pointer(&devCfgMap[0]))
	var controlq *virtio.Queue
	var cursorq *virtio.Queue
	for {
		before := dev.ConfigGeneration()
		dev.Reset()
		needFeats := uint64(virtio.F_VERSION_1 | _VIRTIO_GPU_F_VIRGL)
		if feats := dev.Features(); feats&needFeats != needFeats {
			return nil, fmt.Errorf("gpu: supports features %#x need at least %#x", feats, needFeats)
		}
		if err := dev.NegotiateFeatures(needFeats); err != nil {
			return nil, err
		}
		controlq, err = dev.ConfigureQueue(virtGPUControlQueue)
		if err != nil {
			return nil, err
		}
		cursorq, err = dev.ConfigureQueue(virtGPUCursorQueue)
		if err != nil {
			return nil, err
		}
		if after := dev.ConfigGeneration(); after != before {
			// Configuration changed under us.
			continue
		}
		dev.Start()
		break
	}
	gpu := &Device{
		dev: dev,
	}
	gpu.cfg.cfg = cfg

	gpu.cursor.q = virtio.NewCommander(cursorq)
	cursorBuf, err := virtio.NewIOMem(0, 1e4)
	if err != nil {
		return nil, err
	}
	gpu.cursor.buf = cursorBuf

	gpu.cmd.q = virtio.NewCommander(controlq)
	cmdBuf, err := virtio.NewIOMem(0, 1e7)
	if err != nil {
		return nil, err
	}
	gpu.cmd.buf = cmdBuf
	ch, err := dev.ConfigInterrupt()
	if err != nil {
		return nil, err
	}
	gpu.cfg.notify = ch
	return gpu, nil
}

func (d *Device) ConfigNotify() <-chan struct{} {
	return d.cfg.notify
}

func (d *Device) NewCursor(img *image.RGBA, hotspot image.Point) (Resource, error) {
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	if width != 64 || height != 64 {
		// Qemu rejects any other size.
		return 0, errors.New("virtgpu: cursor dimensions are not 64x64")
	}
	if img.Stride != width*4 {
		return 0, errors.New("virtgpu: cursor stride is not width*4")
	}
	resID := d.cmdResourceCreate2D(VIRGL_FORMAT_B8G8R8A8_SRGB, uint32(width), uint32(height))
	cursor, err := d.alloc(width * height * 4)
	if err != nil {
		return 0, err
	}
	copy(cursor.Mem, img.Pix)
	d.cmdResourceAttachBacking(resID, cursor)
	d.cmdTransferToHost2D(resID, 0, rect{width: uint32(width), height: uint32(height)}, true)
	// Make sure the resource is created before using it.
	if err := d.Flush3D(); err != nil {
		return 0, err
	}
	d.updateCursor(resID)
	return resID, nil
}

func (d *Device) QueryScanout() (int, int, error) {
	if d.cfg.cfg.num_scanouts == 0 {
		return 0, 0, errors.New("gpu: no available scanouts")
	}
	if err := d.updateScanout(); err != nil {
		return 0, 0, err
	}
	return int(d.scanout.rect.width), int(d.scanout.rect.height), nil
}

func (d *Device) newID() uint32 {
	if d.nextID == ^uint32(0) {
		panic("out of id numbers")
	}
	d.nextID++
	return d.nextID
}

func (d *Device) CreateDepthState(enable, mask bool, fun uint32) Handle {
	const cmdLen = 5
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	id := Handle(d.newID())
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_CREATE_OBJECT, VIRGL_OBJECT_DSA))
	bo.PutUint32(cmd[4:8], uint32(id))
	en := uint32(0)
	if enable {
		en = 1
	}
	ma := uint32(0)
	if mask {
		ma = 1
	}
	state := fun<<2 | ma<<1 | en
	bo.PutUint32(cmd[8:12], state)
	d.submit3d(cmd)
	return id
}

func (d *Device) CreateBlend(enable bool, rgbFunc, sfactor, dfactor uint32) Handle {
	const maxColorBufs = 8
	const cmdLen = maxColorBufs + 3
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	id := Handle(d.newID())
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_CREATE_OBJECT, VIRGL_OBJECT_BLEND))
	bo.PutUint32(cmd[4:8], uint32(id))
	bo.PutUint32(cmd[8:12], 0 /* S0, unused */)
	bo.PutUint32(cmd[12:16], 0 /* S1, unused */)
	en := uint32(0)
	if enable {
		en = 1
	}
	alphaDstFactor := dfactor
	alphaSrcFactor := sfactor
	alphaFunc := rgbFunc
	blendState := PIPE_MASK_RGBA<<27 | alphaDstFactor<<22 | alphaSrcFactor<<17 | alphaFunc<<14 | dfactor<<9 | sfactor<<4 | rgbFunc<<1 | en
	bo.PutUint32(cmd[16:20], blendState)
	d.submit3d(cmd)
	return id
}

func (d *Device) CreateSurface(res Resource, format uint32) Handle {
	const cmdLen = 5
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	id := Handle(d.newID())
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_CREATE_OBJECT, _VIRGL_OBJECT_SURFACE))
	bo.PutUint32(cmd[4:8], uint32(id))
	bo.PutUint32(cmd[8:12], uint32(res))
	bo.PutUint32(cmd[12:16], format)
	bo.PutUint32(cmd[16:20], 0 /* First element */)
	bo.PutUint32(cmd[20:24], 0 /* Last element */)
	d.submit3d(cmd)
	return id
}

func (d *Device) CreateSamplerState(wrapS, wrapT, minImg, minMip, magImg uint32) Handle {
	const cmdLen = 9
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	id := Handle(d.newID())
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_CREATE_OBJECT, _VIRGL_OBJECT_SAMPLER_STATE))
	bo.PutUint32(cmd[4:8], uint32(id))
	state := magImg<<13 | minMip<<11 | minImg<<9 | wrapT<<3 | wrapS
	bo.PutUint32(cmd[8:12], state)
	d.submit3d(cmd)
	return id
}

func (d *Device) CreateSamplerView(res Resource, format, target, swizzle uint32) Handle {
	const cmdLen = 6
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	id := Handle(d.newID())
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_CREATE_OBJECT, _VIRGL_OBJECT_SAMPLER_VIEW))
	bo.PutUint32(cmd[4:8], uint32(id))
	bo.PutUint32(cmd[8:12], uint32(res))
	bo.PutUint32(cmd[12:16], target<<24|format)
	bo.PutUint32(cmd[16:20], 0 /* VIRGL_OBJ_SAMPLER_VIEW_BUFFER_FIRST_ELEMENT */)
	bo.PutUint32(cmd[20:24], 0 /* VIRGL_OBJ_SAMPLER_VIEW_BUFFER_LAST_ELEMENT */)
	bo.PutUint32(cmd[24:28], swizzle)
	d.submit3d(cmd)
	return id
}

func (d *Device) CreateShader(typ uint32, src string) Handle {
	const headerLen = 5
	src = src + "\x00"
	// Round up source length.
	cmdLen := headerLen + (len(src)+3)/4
	if cmdLen != int(uint16(cmdLen)) {
		panic(fmt.Errorf("gpu: shader source too big (%d)", len(src)))
	}
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	id := Handle(d.newID())
	bo.PutUint32(cmd[0:4], encodeCmdHeader(uint16(cmdLen), _VIRGL_CCMD_CREATE_OBJECT, _VIRGL_OBJECT_SHADER))
	bo.PutUint32(cmd[4:8], uint32(id))
	bo.PutUint32(cmd[8:12], typ)
	bo.PutUint32(cmd[12:16], 0 /* offlen */)
	// TODO: We should be passing in the exact number of tokens in the
	// shader source. That's difficult to determine without a parser,
	// so let's hope a big count is enough.
	bo.PutUint32(cmd[16:20], 10000 /* num tokens */)
	bo.PutUint32(cmd[20:24], 0 /* num vertex outputs */)
	copy(cmd[24:], src)
	d.submit3d(cmd)
	return id
}

func (d *Device) Copy(res Resource, data []byte, width, height int) {
	// Transfer to a staging resource and do a synchronous copy to the
	// destination.
	staging := d.CmdResourceCreate3D(ResourceCreate3DReq{
		Width:      uint32(len(data)),
		Height:     1,
		Depth:      1,
		Array_size: 1,
		Bind:       _VIRGL_BIND_STAGING,
	})

	d.CmdCtxAttachResource(staging)
	defer d.CmdCtxDetachResource(staging)
	defer d.CmdResourceUnref(staging)

	s, err := d.alloc(len(data))
	if err != nil {
		d.setErr(err)
		return
	}
	copy(s.Mem, data)
	d.cmdResourceAttachBacking(staging, s)
	d.copyTransfer3D(res, width, height, staging, 0)
}

func (d *Device) copyTransfer3D(dst Resource, dwidth, dheight int, src Resource, soff int) {
	const cmdLen = 14
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(uint16(cmdLen), _VIRGL_CCMD_COPY_TRANSFER3D, 0))
	bo.PutUint32(cmd[4:8], uint32(dst))
	bo.PutUint32(cmd[8:12], 0 /* Level */)
	bo.PutUint32(cmd[12:16], 0 /* Usage */)
	bo.PutUint32(cmd[16:20], 0 /* Stride */)
	bo.PutUint32(cmd[20:24], 0 /* Layer stride */)
	bo.PutUint32(cmd[24:28], 0 /* X */)
	bo.PutUint32(cmd[28:32], 0 /* Y */)
	bo.PutUint32(cmd[32:36], 0 /* Z */)
	bo.PutUint32(cmd[36:40], uint32(dwidth) /* Width */)
	bo.PutUint32(cmd[40:44], uint32(dheight) /* Height */)
	bo.PutUint32(cmd[44:48], 1 /* Depth */)
	bo.PutUint32(cmd[48:52], uint32(src) /* Source */)
	bo.PutUint32(cmd[52:56], uint32(soff) /* Source offset */)
	bo.PutUint32(cmd[56:60], 1 /* Synchronized */)
	d.submit3d(cmd)
}

func (d *Device) resourceInlineWrite(res Resource, data []byte, width, height int) error {
	const headerLen = 11
	// Compute total command length, rounding up the data length.
	cmdLen := headerLen + (len(data)+3)/4
	if cmdLen != int(uint16(cmdLen)) {
		return fmt.Errorf("gpu: data too big (%d bytes) for inline copy", len(data))
	}
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(uint16(cmdLen), _VIRGL_CCMD_RESOURCE_INLINE_WRITE, 0))
	bo.PutUint32(cmd[4:8], uint32(res))
	bo.PutUint32(cmd[8:12], 0 /* Level */)
	bo.PutUint32(cmd[12:16], 0 /* Usage */)
	bo.PutUint32(cmd[16:20], 0 /* Stride */)
	bo.PutUint32(cmd[20:24], 0 /* Layer stride */)
	bo.PutUint32(cmd[24:28], 0 /* X */)
	bo.PutUint32(cmd[28:32], 0 /* Y */)
	bo.PutUint32(cmd[32:36], 0 /* Z */)
	bo.PutUint32(cmd[36:40], uint32(width) /* Width */)
	bo.PutUint32(cmd[40:44], uint32(height) /* Height */)
	bo.PutUint32(cmd[44:48], 1 /* Depth */)
	if n := copy(cmd[48:], data); n != len(data) {
		panic("gpu: wrong command size")
	}
	d.submit3d(cmd)
	return nil
}

func (d *Device) DestroyObject(h Handle) {
	const cmdLen = 1
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_DESTROY_OBJECT, 0))
	bo.PutUint32(cmd[4:8], uint32(h))
	d.submit3d(cmd)
}

func (d *Device) BindObject(typ uint8, h Handle) {
	const cmdLen = 1
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_BIND_OBJECT, typ))
	bo.PutUint32(cmd[4:8], uint32(h))
	d.submit3d(cmd)
}

func (d *Device) Viewport(x, y, width, height int) {
	const cmdLen = 7
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_SET_VIEWPORT_STATE, 0))
	bo.PutUint32(cmd[4:8], 0 /* Start slot */)
	bo.PutUint32(cmd[8:12], math.Float32bits(.5*float32(width)) /* Scale X */)
	bo.PutUint32(cmd[12:16], math.Float32bits(.5*float32(height)) /* Scale Y */)
	bo.PutUint32(cmd[16:20], math.Float32bits(1.0) /* Scale Z */)
	bo.PutUint32(cmd[20:24], math.Float32bits(float32(x)+.5*float32(width)) /* Translate X */)
	bo.PutUint32(cmd[24:28], math.Float32bits(float32(y)+.5*float32(height)) /* Translate Y */)
	bo.PutUint32(cmd[28:32], math.Float32bits(0.0) /* Translate Z */)
	d.submit3d(cmd)
}

func (d *Device) BindShader(typ uint32, shader Handle) {
	const cmdLen = 2
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_BIND_SHADER, 0))
	bo.PutUint32(cmd[4:8], uint32(shader))
	bo.PutUint32(cmd[8:12], typ)
	d.submit3d(cmd)
}

func (d *Device) SetSamplerViews(shaderType uint32, views []Handle) {
	cmdLen := 2 + len(views)
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(uint16(cmdLen), _VIRGL_CCMD_SET_SAMPLER_VIEWS, 0))
	bo.PutUint32(cmd[4:8], shaderType)
	bo.PutUint32(cmd[8:12], 0 /* Start slot */)
	for i, view := range views {
		bo.PutUint32(cmd[12+i*4:], uint32(view))
	}
	d.submit3d(cmd)
}

func (d *Device) SetSamplerStates(shaderType uint32, states []Handle) {
	cmdLen := 2 + len(states)
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(uint16(cmdLen), _VIRGL_CCMD_BIND_SAMPLER_STATES, 0))
	bo.PutUint32(cmd[4:8], shaderType)
	bo.PutUint32(cmd[8:12], 0 /* Start slot */)
	for i, state := range states {
		bo.PutUint32(cmd[12+i*4:], uint32(state))
	}
	d.submit3d(cmd)
}

func (d *Device) SetUniformBuffer(shaderType, index, offset, length uint32, buf Resource) {
	const cmdLen = 5
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_SET_UNIFORM_BUFFER, 0))
	bo.PutUint32(cmd[4:8], shaderType)
	bo.PutUint32(cmd[8:12], index)
	bo.PutUint32(cmd[12:16], offset)
	bo.PutUint32(cmd[16:20], length)
	bo.PutUint32(cmd[20:24], uint32(buf))
	d.submit3d(cmd)
}

func (d *Device) SetIndexBuffer(buffer Resource, elemSize, offset uint32) {
	const cmdLen = 3
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(uint16(cmdLen), _VIRGL_CCMD_SET_INDEX_BUFFER, 0))
	bo.PutUint32(cmd[4:8], uint32(buffer))
	bo.PutUint32(cmd[8:12], elemSize)
	bo.PutUint32(cmd[12:16], offset)
	d.submit3d(cmd)
}

func (d *Device) SetVertexBuffers(buffers []VertexBuffer) {
	cmdLen := 3 * len(buffers)
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(uint16(cmdLen), _VIRGL_CCMD_SET_VERTEX_BUFFERS, 0))
	for i, buf := range buffers {
		off := 4 + i*4*3
		bo.PutUint32(cmd[off:], buf.Stride)
		bo.PutUint32(cmd[off+4:], buf.Offset)
		bo.PutUint32(cmd[off+8:], uint32(buf.Buffer))
	}
	d.submit3d(cmd)
}

func (d *Device) CreateVertexElements(elems []VertexElement) (Handle, error) {
	cmdLen := 1 + len(elems)*4
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	id := Handle(d.newID())
	bo.PutUint32(cmd[0:4], encodeCmdHeader(uint16(cmdLen), _VIRGL_CCMD_CREATE_OBJECT, VIRGL_OBJECT_VERTEX_ELEMENTS))
	bo.PutUint32(cmd[4:8], uint32(id))
	for i, elem := range elems {
		off := 8 + i*4*4
		bo.PutUint32(cmd[off:], elem.Offset)
		bo.PutUint32(cmd[off+4:], elem.Divisor)
		bo.PutUint32(cmd[off+8:], elem.BufferIndex)
		bo.PutUint32(cmd[off+12:], elem.Format)
	}
	d.submit3d(cmd)
	return id, nil
}

func (d *Device) SetFramebufferState(colorSurf, depthSurf Handle) {
	const cmdLen = 3
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_SET_FRAMEBUFFER_STATE, 0))
	bo.PutUint32(cmd[4:8], 1 /* Number of color buffers */)
	bo.PutUint32(cmd[8:12], uint32(depthSurf))
	bo.PutUint32(cmd[12:16], uint32(colorSurf))
	d.submit3d(cmd)
}

func (d *Device) Draw(indexed bool, mode, offset, count uint32) {
	const cmdLen = 12
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_DRAW_VBO, 0))
	bo.PutUint32(cmd[4:8], offset)
	bo.PutUint32(cmd[8:12], count)
	bo.PutUint32(cmd[12:16], mode)
	ind := 0
	if indexed {
		ind = 1
	}
	bo.PutUint32(cmd[16:20], uint32(ind))
	bo.PutUint32(cmd[20:24], 0 /* Instance count */)
	bo.PutUint32(cmd[24:28], 0 /* Index bias */)
	bo.PutUint32(cmd[28:32], 0 /* Start instance */)
	bo.PutUint32(cmd[32:36], 0 /* Primitive restart */)
	bo.PutUint32(cmd[36:40], 0 /* Restart index */)
	bo.PutUint32(cmd[40:44], 0 /* Min index */)
	bo.PutUint32(cmd[44:48], ^uint32(0) /* Max index */)
	d.submit3d(cmd)
}

func (d *Device) Clear(buffers uint32, color [4]float32, depth float32) {
	const cmdLen = 8
	cmd := make([]byte, 4+cmdLen*4)
	bo := binary.LittleEndian
	bo.PutUint32(cmd[0:4], encodeCmdHeader(cmdLen, _VIRGL_CCMD_CLEAR, 0))
	bo.PutUint32(cmd[4:8], buffers)
	for i, c := range color {
		bo.PutUint32(cmd[8+i*4:], math.Float32bits(c))
	}
	bo.PutUint64(cmd[24:24+8], math.Float64bits(float64(depth)))
	d.submit3d(cmd)
}

func encodeCmdHeader(size uint16, typ uint8, subtype uint8) uint32 {
	return uint32(size)<<16 | uint32(subtype)<<8 | uint32(typ)
}

func (d *Device) submit3d(cmd []byte) {
	total := len(cmd)
	// Make sure there is room for the 3d command response.
	total += int(unsafe.Sizeof(ctrlHdr{}))
	if !d.cmd3d.begun {
		total += int(unsafe.Sizeof(cmdSubmitReq{}))
	}
	d.ensure(total)
	if !d.cmd3d.begun {
		d.cmd3d.size = 0
		d.cmd3d.offset = d.cmd.bufOff
		// Write 3d command header.
		var req *cmdSubmitReq
		header, err := d.alloc(int(unsafe.Sizeof(*req)))
		if err != nil {
			// We ensure'd it above.
			panic("no reserved space")
		}
		req = (*cmdSubmitReq)(unsafe.Pointer(&header.Mem[0]))
		*req = cmdSubmitReq{
			hdr: ctrlHdr{
				_type:  _VIRTIO_GPU_CMD_SUBMIT_3D,
				ctx_id: d.ctxID,
			},
		}
		d.cmd3d.begun = true
	}
	buf, err := d.alloc(len(cmd))
	if err != nil {
		// We ensure'd it above.
		panic("no reserved space")
	}
	d.cmd3d.size += len(cmd)
	copy(buf.Mem, cmd)
}

func (d *Device) flush3D() {
	if !d.cmd3d.begun {
		return
	}

	// Initialize command size.
	buf := d.cmd.buf.Mem
	buf = buf[d.cmd3d.offset:]
	// Count is just after the submit 3d header.
	buf = buf[unsafe.Sizeof(ctrlHdr{}):]
	binary.LittleEndian.PutUint32(buf, uint32(d.cmd3d.size))

	// Setup and submit command.
	reqBuf := d.cmd.buf.Slice(d.cmd3d.offset, d.cmd.bufOff)
	respBuf, err := d.alloc(int(unsafe.Sizeof(ctrlHdr{})))
	if err != nil {
		// This is an internal error because submit3d reserves space
		// for the response.
		panic("no reserved space for 3d command submit response")
	}
	d.command(reqBuf, respBuf)
	d.cmd3d.begun = false
}

func (d *Device) sync() {
	d.flush3D()
	d.cmd.q.Sync()
	d.cmd.bufOff = 0
	d.cmd3d.begun = false
}

func (d *Device) Flush3D() error {
	d.sync()
	return d.submitErr
}

func (d *Device) CmdResourceFlush(res Resource) {
	d.cmdResourceFlush(res, d.scanout.rect)
}

func (d *Device) ensure(size int) {
	if d.cmd.bufOff+size > cap(d.cmd.buf.Mem) {
		d.sync()
	}
}

func (d *Device) alloc(size int) (virtio.IOMem, error) {
	d.ensure(size)
	if err := d.cmd.buf.Ensure(d.cmd.bufOff + size); err != nil {
		return virtio.IOMem{}, err
	}
	s := d.cmd.buf.Slice(d.cmd.bufOff, d.cmd.bufOff+size)
	d.cmd.bufOff += size
	return s, nil
}

func (d *Device) allocCommand(reqSize, respSize uintptr) ([2]virtio.IOMem, [2]unsafe.Pointer, bool) {
	d.flush3D()
	var bufs [2]virtio.IOMem
	var ptrs [2]unsafe.Pointer
	var err error
	bufs[0], err = d.alloc(int(reqSize))
	if err != nil {
		d.setErr(err)
		return bufs, ptrs, false
	}
	reqMem := bufs[0].Mem[:reqSize]
	ptrs[0] = unsafe.Pointer(&reqMem[0])
	bufs[1], err = d.alloc(int(respSize))
	if err != nil {
		d.setErr(err)
		return bufs, ptrs, false
	}
	respMem := bufs[1].Mem[:respSize]
	ptrs[1] = unsafe.Pointer(&respMem[0])
	return bufs, ptrs, true
}

func (d *Device) setErr(err error) {
	if d.submitErr == nil {
		d.submitErr = err
	}
}

func (d *Device) command(req, resp virtio.IOMem) {
	for !d.cmd.q.Command(req, resp) {
		_, err := d.cmd.q.Read()
		if err != nil {
			d.setErr(err)
			return
		}
	}
}

func (d *Device) MoveCursor(cursor Resource, x, y uint32) {
	d.cursorCmd(_VIRTIO_GPU_CMD_MOVE_CURSOR, cursor, x, y)
}

func (d *Device) updateCursor(res Resource) {
	d.cursorCmd(_VIRTIO_GPU_CMD_UPDATE_CURSOR, res, 0, 0)
}

func (d *Device) cursorCmd(cmd uint32, res Resource, x, y uint32) {
	var req *updateCursorReq
	var resp *ctrlHdr
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return
	}
	req = (*updateCursorReq)(ptrs[0])
	*req = updateCursorReq{
		hdr: ctrlHdr{
			_type: cmd,
		},
		pos: cursorPos{
			scanout_id: d.scanout.id,
			x:          x,
			y:          y,
		},
		resource_id: res,
	}
	for !d.cursor.q.Command(bufs[0], bufs[1]) {
		_, err := d.cursor.q.Read()
		if err != nil {
			d.setErr(err)
			return
		}
	}
}

func (d *Device) cmdCtxCreate() uint32 {
	var req *ctxCreateReq
	var resp *ctrlHdr
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return 0
	}
	ctxID := d.newID()
	req = (*ctxCreateReq)(ptrs[0])
	name := fmt.Sprintf("gpu%d", ctxID)
	*req = ctxCreateReq{
		hdr: ctrlHdr{
			_type:  _VIRTIO_GPU_CMD_CTX_CREATE,
			ctx_id: ctxID,
		},
		nlen: uint32(len(name)),
	}
	copy(req.debug_name[:], name)
	d.command(bufs[0], bufs[1])
	return ctxID
}

func (d *Device) cmdResourceAttachBacking(res Resource, buffer virtio.IOMem) {
	nentries := len(buffer.Blocks)
	var req *resourceAttachBackingReq
	var resp *ctrlHdr
	hdrSize := unsafe.Sizeof(*req)
	reqSize := hdrSize + uintptr(nentries)*unsafe.Sizeof(memEntry{})
	bufs, ptrs, ok := d.allocCommand(reqSize, unsafe.Sizeof(*resp))
	if !ok {
		return
	}
	req = (*resourceAttachBackingReq)(ptrs[0])
	entries := ((*[1 << 30]memEntry)(unsafe.Pointer(uintptr(ptrs[0]) + hdrSize)))[:nentries]
	for i, page := range buffer.Blocks {
		entries[i].addr = uint64(page.Addr)
		entries[i].length = uint32(page.Size)
	}
	*req = resourceAttachBackingReq{
		hdr: ctrlHdr{
			_type: _VIRTIO_GPU_CMD_RESOURCE_ATTACH_BACKING,
		},
		resource_id: res,
		nr_entries:  uint32(nentries),
	}
	d.command(bufs[0], bufs[1])
}

func (d *Device) cmdResourceDetachBacking(res Resource) {
	var req *resourceDetachBackingReq
	var resp *ctrlHdr
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return
	}
	req = (*resourceDetachBackingReq)(ptrs[0])
	*req = resourceDetachBackingReq{
		hdr: ctrlHdr{
			_type: _VIRTIO_GPU_CMD_RESOURCE_DETACH_BACKING,
		},
		resource_id: res,
	}
	d.command(bufs[0], bufs[1])
}

func (d *Device) cmdResourceFlush(res Resource, r rect) {
	var req *resourceFlushReq
	var resp *ctrlHdr
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return
	}
	req = (*resourceFlushReq)(ptrs[0])
	*req = resourceFlushReq{
		hdr: ctrlHdr{
			_type: _VIRTIO_GPU_CMD_RESOURCE_FLUSH,
		},
		r:           r,
		resource_id: res,
	}
	d.command(bufs[0], bufs[1])
}

func (d *Device) cmdTransferToHost2D(res Resource, off uint64, r rect, fence bool) {
	var req *transferToHost2DReq
	var resp *ctrlHdr
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return
	}
	req = (*transferToHost2DReq)(ptrs[0])
	*req = transferToHost2DReq{
		hdr: ctrlHdr{
			_type: _VIRTIO_GPU_CMD_TRANSFER_TO_HOST_2D,
		},
		r:           r,
		offset:      off,
		resource_id: res,
	}
	if fence {
		req.hdr.flags |= _VIRTIO_GPU_FLAG_FENCE
		req.hdr.fence_id = 1 // Don't care, but hangs if 0.
	}
	d.command(bufs[0], bufs[1])
}

func (d *Device) CmdCtxDetachResource(resID Resource) {
	var req *ctxResourceReq
	var resp *ctrlHdr
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return
	}
	req = (*ctxResourceReq)(ptrs[0])
	*req = ctxResourceReq{
		hdr: ctrlHdr{
			_type:  _VIRTIO_GPU_CMD_CTX_DETACH_RESOURCE,
			ctx_id: d.ctxID,
		},
		resource_id: resID,
	}
	d.command(bufs[0], bufs[1])
}

func (d *Device) CmdCtxAttachResource(resID Resource) {
	var req *ctxResourceReq
	var resp *ctrlHdr
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return
	}
	req = (*ctxResourceReq)(ptrs[0])
	*req = ctxResourceReq{
		hdr: ctrlHdr{
			_type:  _VIRTIO_GPU_CMD_CTX_ATTACH_RESOURCE,
			ctx_id: d.ctxID,
		},
		resource_id: resID,
	}
	d.command(bufs[0], bufs[1])
}

func (d *Device) CmdSetScanout(res Resource) {
	var req *setScanoutReq
	var resp *ctrlHdr
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return
	}
	req = (*setScanoutReq)(ptrs[0])
	*req = setScanoutReq{
		hdr: ctrlHdr{
			_type: _VIRTIO_GPU_CMD_SET_SCANOUT,
		},
		r:           d.scanout.rect,
		scanout_id:  d.scanout.id,
		resource_id: res,
	}
	d.command(bufs[0], bufs[1])
}

func (d *Device) CmdResourceUnref(resID Resource) {
	var req *resourceUnrefReq
	var resp *ctrlHdr
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return
	}
	req = (*resourceUnrefReq)(ptrs[0])
	*req = resourceUnrefReq{
		hdr: ctrlHdr{
			_type: _VIRTIO_GPU_CMD_RESOURCE_UNREF,
		},
		resource_id: resID,
	}
	d.command(bufs[0], bufs[1])
}

func (d *Device) cmdResourceCreate2D(format, width, height uint32) Resource {
	var req *resourceCreate2DReq
	var resp *ctrlHdr
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return 0
	}
	resID := Resource(d.newID())
	req = (*resourceCreate2DReq)(ptrs[0])
	*req = resourceCreate2DReq{
		hdr: ctrlHdr{
			_type: _VIRTIO_GPU_CMD_RESOURCE_CREATE_2D,
		},
		resource_id: resID,
		format:      format,
		width:       width,
		height:      height,
	}
	d.command(bufs[0], bufs[1])
	return resID
}

func (d *Device) CmdResourceCreate3D(cmd ResourceCreate3DReq) Resource {
	var req *ResourceCreate3DReq
	var resp *ctrlHdr
	cmd.resource_id = Resource(d.newID())
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(*req), unsafe.Sizeof(*resp))
	if !ok {
		return 0
	}
	req = (*ResourceCreate3DReq)(ptrs[0])
	*req = cmd
	req.hdr._type = _VIRTIO_GPU_CMD_RESOURCE_CREATE_3D
	d.command(bufs[0], bufs[1])
	return cmd.resource_id
}

func (d *Device) queryCaps() (capsV2, error) {
	inf, err := d.cmdGetCapsetInfo(1)
	if err != nil {
		return capsV2{}, err
	}
	const capVer = 2
	if v := inf.capset_max_version; v < capVer {
		return capsV2{}, fmt.Errorf("virtgpu: VIRTIO_GPU_CAPSET_VIRGL2 version %d, expected at least %d", v, capVer)
	}

	// Request capset v2.
	respSize := unsafe.Sizeof(ctrlHdr{}) + uintptr(inf.capset_max_size)
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(getCapsetReq{}), respSize)
	if !ok {
		return capsV2{}, d.submitErr
	}
	req := (*getCapsetReq)(ptrs[0])
	resp := (*ctrlHdr)(ptrs[1])
	capsBuf := bufs[1].Mem[unsafe.Sizeof(ctrlHdr{}):]
	if uintptr(len(capsBuf)) < unsafe.Sizeof(capsV2{}) {
		return capsV2{}, fmt.Errorf("virtgpu: the VIRTIO_GPU_CAPSET_VIRGL2 capability structure is too small")
	}
	cap := (*capsV2)(unsafe.Pointer(&capsBuf[0]))
	*req = getCapsetReq{
		hdr: ctrlHdr{
			_type: _VIRTIO_GPU_CMD_GET_CAPSET,
		},
		capset_id:      inf.capset_id,
		capset_version: capVer,
	}
	d.command(bufs[0], bufs[1])
	d.sync()
	if c := resp._type; c != _VIRTIO_GPU_RESP_OK_CAPSET {
		return capsV2{}, fmt.Errorf("virtgpu: invalid VIRTIO_GPU_CMD_GET_CAPSET response: %#x", c)
	}
	return *cap, nil
}

func (d *Device) cmdGetCapsetInfo(idx int) (capsetInfoResp, error) {
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(getCapsetInfoReq{}), unsafe.Sizeof(capsetInfoResp{}))
	if !ok {
		return capsetInfoResp{}, d.submitErr
	}
	req := (*getCapsetInfoReq)(ptrs[0])
	resp := (*capsetInfoResp)(ptrs[1])
	*req = getCapsetInfoReq{
		hdr: ctrlHdr{
			_type: _VIRTIO_GPU_CMD_GET_CAPSET_INFO,
		},
		capset_index: uint32(idx),
	}
	d.command(bufs[0], bufs[1])
	d.sync()
	if c := resp.hdr._type; c != _VIRTIO_GPU_RESP_OK_CAPSET_INFO {
		return capsetInfoResp{}, fmt.Errorf("virtgpu: invalid VIRTIO_GPU_CMD_GET_CAPSET_INFO response: %#x", c)
	}
	return *resp, nil
}

func (d *Device) updateScanout() error {
	// Request display info.
	bufs, ptrs, ok := d.allocCommand(unsafe.Sizeof(ctrlHdr{}), unsafe.Sizeof(displayInfoResp{}))
	if !ok {
		return d.submitErr
	}
	req := (*ctrlHdr)(ptrs[0])
	*req = ctrlHdr{
		_type: _VIRTIO_GPU_CMD_GET_DISPLAY_INFO,
	}
	d.command(bufs[0], bufs[1])
	d.sync()
	resp := (*displayInfoResp)(ptrs[1])
	if c := resp.hdr._type; c != _VIRTIO_GPU_RESP_OK_DISPLAY_INFO {
		return fmt.Errorf("virtgpu: invalid VIRTIO_GPU_CMD_GET_DISPLAY_INFO response: %#x", c)
	}

	// Use the first enabled scanout, if any
	d.scanout.id = 0
	d.scanout.rect = rect{}
	for i, mode := range resp.pmodes {
		if mode.enabled != 0 {
			d.scanout.id = uint32(i)
			d.scanout.rect = mode.r
			break
		}
	}
	return nil
}
