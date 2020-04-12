// SPDX-License-Identifier: Unlicense OR MIT

// Package input implements drivers for virtio input devices.
package input

import (
	"bytes"
	"errors"
	"fmt"
	"unsafe"

	"eliasnaur.com/unik/kernel"
	"eliasnaur.com/unik/virtio"
)

type Device struct {
	Name string

	dev *virtio.Device
	cfg *config
	r   *virtio.Reader
	buf []byte
	off int
}

type Event struct {
	Type  uint16
	Code  uint16
	Value uint32
}

type AbsInfo struct {
	Min  uint32
	Max  uint32
	Fuzz uint32
	Flat uint32
	Res  uint32
}

type config struct {
	_select  uint8
	subsel   uint8
	size     uint8
	reserved [5]uint8
	subcfg   [128]byte
}

const (
	EV_SYN = 0x00
	EV_KEY = 0x01
	EV_REL = 0x02
	EV_ABS = 0x03
)

const (
	BTN_LEFT   = 0x110
	BTN_RIGHT  = 0x111
	BTN_MIDDLE = 0x112
	BTN_WHEEL  = 0x150
)

const (
	REL_WHEEL  = 0x08
	REL_HWHEEL = 0x06

	ABS_X = 0x00
	ABS_Y = 0x01
)

const (
	_VIRTIO_INPUT_CFG_UNSET     = 0x00
	_VIRTIO_INPUT_CFG_ID_NAME   = 0x01
	_VIRTIO_INPUT_CFG_ID_SERIAL = 0x02
	_VIRTIO_INPUT_CFG_ID_DEVIDS = 0x03
	_VIRTIO_INPUT_CFG_PROP_BITS = 0x10
	_VIRTIO_INPUT_CFG_EV_BITS   = 0x11
	_VIRTIO_INPUT_CFG_ABS_INFO  = 0x12
)

const virtInputEventQueue = 0

const eventSize = int(unsafe.Sizeof(Event{}))

func New() (*Device, error) {
	const deviceTypeInput = 18
	vdev, err := virtio.New(deviceTypeInput)
	if err != nil {
		return nil, err
	}
	return newDevice(vdev)
}

func newDevice(dev *virtio.Device) (*Device, error) {
	devCfgMap, _, err := dev.FindCapability(virtio.PCI_CAP_DEVICE_CFG)
	if err != nil {
		return nil, err
	}
	if unsafe.Sizeof(config{}) > uintptr(len(devCfgMap)) {
		return nil, errors.New("gpu: device configuration area too small")
	}
	d := &Device{
		dev: dev,
		cfg: (*config)(unsafe.Pointer(&devCfgMap[0])),
	}
	var eventq *virtio.Queue
	for {
		before := dev.ConfigGeneration()
		dev.Reset()
		needFeats := uint64(virtio.F_VERSION_1)
		if feats := dev.Features(); feats&needFeats != needFeats {
			return nil, fmt.Errorf("input: supports features %#x need at least %#x", feats, needFeats)
		}
		if err := dev.NegotiateFeatures(needFeats); err != nil {
			return nil, err
		}
		if name, ok := d.cfg.queryCfg(_VIRTIO_INPUT_CFG_ID_NAME, 0); ok {
			// The virtio specs says that strings do not include a
			// trailing NUL. But Qemu does have them.
			name = bytes.TrimRight(name, "\x00")
			d.Name = string(name)
		}
		eventq, err = dev.ConfigureQueue(virtInputEventQueue)
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
	bufSize := int(eventq.Size() * eventSize)
	dmaBuf, err := virtio.NewIOMem(bufSize, bufSize)
	if err != nil {
		return nil, err
	}
	d.r, err = virtio.NewReader(eventq, dmaBuf.Slice(0, bufSize), eventSize)
	if err != nil {
		return nil, err
	}
	d.buf = make([]byte, bufSize)
	return d, nil
}

func (d *Device) AbsInfo(axis uint8) (AbsInfo, error) {
	abs, ok := d.cfg.queryCfg(_VIRTIO_INPUT_CFG_ABS_INFO, axis)
	if !ok {
		return AbsInfo{}, errors.New("input: no axis information")
	}
	if got, exp := len(abs), int(unsafe.Sizeof(AbsInfo{})); got < exp {
		return AbsInfo{}, fmt.Errorf("input: axis info truncated to %d, expected %d", got, exp)
	}
	return *(*AbsInfo)(unsafe.Pointer(&abs[0])), nil
}

func (d *Device) Read(events []Event) (int, error) {
	nevents := 0
	var err error
	if d.off < eventSize {
		var n int
		n, err = d.r.Read(d.buf[d.off:])
		d.off += n
	}
	buf := d.buf[:d.off]
	for len(buf) >= eventSize && len(events) > 0 {
		event := (*Event)(unsafe.Pointer(&buf[0]))
		events[0] = *event
		events = events[1:]
		buf = buf[eventSize:]
		nevents++
	}
	d.off = copy(d.buf, buf)
	return nevents, err
}

func (c *config) queryCfg(_select, subsel uint8) ([]byte, bool) {
	// Query name.
	kernel.StoreUint8(&c._select, _select)
	kernel.StoreUint8(&c.subsel, subsel)
	size := kernel.LoadUint8(&c.size)
	if size == 0 {
		return nil, false
	}
	sub := make([]byte, size)
	copy(sub, c.subcfg[:size])
	return sub, true
}
