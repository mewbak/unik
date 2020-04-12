// SPDX-License-Identifier: Unlicense OR MIT

package virtio

import (
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"syscall"
	"unsafe"

	"eliasnaur.com/unik/kernel"
	"eliasnaur.com/unik/pci"
)

type Device struct {
	addr       pci.Address
	cfg        *virtioConfig
	interrupts struct {
		table          *pci.InterruptTable
		usedInterrupts int
	}
	notify struct {
		base       []byte
		multiplier uint32
	}
}

type virtioConfig struct {
	device_feature_select uint32
	device_feature        uint32
	driver_feature_select uint32
	driver_feature        uint32
	msix_vector           uint16
	num_queues            uint16
	device_status         uint8
	config_generation     uint8

	queue_select      uint16
	queue_size        uint16
	queue_msix_vector uint16
	queue_enable      uint16
	queue_notify_off  uint16
	queue_desc        uint64
	queue_driver      uint64
	queue_device      uint64
}

type Queue struct {
	queue      *virtioDeviceQueue
	size       uint16
	notifyAddr *uint16
	queueIndex uint16
	interrupt  <-chan struct{}
}

type virtioDeviceQueue struct {
	descriptors [maxQueueSize]queueDescriptor
	available   struct {
		flags      uint16
		idx        uint16
		ring       [maxQueueSize]uint16
		used_event uint16
	}
	used struct {
		flags uint16
		idx   uint16
		ring  [maxQueueSize]struct {
			id  uint32
			len uint32
		}
		avail_event uint16
	}
}

type queueDescriptor struct {
	addr  uint64
	len   uint32
	flags uint16
	next  uint16
}

// PhysMem is a memory region backed by physical
// memory pages.
type IOMem struct {
	Mem    []byte
	Blocks []PhysPage
}

type PhysPage struct {
	Addr uintptr
	Size int
}

const (
	_PCI_CAP_ID_VNDR = 0x9

	F_VERSION_1 = 1 << 32

	// PCI capabilities.
	_VIRTIO_PCI_CAP_COMMON_CFG = 1
	_VIRTIO_PCI_CAP_NOTIFY_CFG = 2
	_VIRTIO_PCI_CAP_ISR_CFG    = 3
	PCI_CAP_DEVICE_CFG         = 4
	_VIRTIO_PCI_CAP_PCI_CFG    = 5

	// Device status flags.
	_ACKNOWLEDGE        = 1
	_DRIVER             = 2
	_FAILED             = 128
	_FEATURES_OK        = 8
	_DRIVER_OK          = 4
	_DEVICE_NEEDS_RESET = 64

	// Descriptor flags.
	_VIRTQ_DESC_F_NEXT     = 1
	_VIRTQ_DESC_F_WRITE    = 2
	_VIRTQ_DESC_F_INDIRECT = 4

	// maxQueueSize is the maximum queue size. Must be a power of 2.
	maxQueueSize = 1 << 7
)

func New(function int) (*Device, error) {
	addrs, err := pci.Detect()
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		if a.ReadVendorID() != 0x1af4 {
			// Not a Virtio device.
			continue
		}

		devID := a.ReadDeviceID()
		if !(0x1040 <= devID && devID <= 0x107f) {
			// Not a virtio device, or a legacy virtio device.
			continue
		}
		if f := int(devID - 0x1040); f != function {
			// Not the correct type.
			continue
		}
		return newVirtioDevice(a)
	}
	return nil, fmt.Errorf("virtio: no virtio device type %d", function)
}

func newVirtioDevice(addr pci.Address) (*Device, error) {
	dev := &Device{
		addr: addr,
	}
	cfg, _, err := dev.FindCapability(_VIRTIO_PCI_CAP_COMMON_CFG)
	if err != nil {
		return nil, err
	}
	if unsafe.Sizeof(virtioConfig{}) > uintptr(len(cfg)) {
		return nil, errors.New("virtio: common configuration area too small")
	}
	dev.cfg = (*virtioConfig)(unsafe.Pointer(&cfg[0]))
	notify, offset, err := dev.FindCapability(_VIRTIO_PCI_CAP_NOTIFY_CFG)
	if err != nil {
		return nil, err
	}
	// The notify_off_multiplier field is located just after the
	// capability structure.
	multiplier := addr.ReadPCIRegister(offset + 16)
	dev.notify.base = notify
	dev.notify.multiplier = multiplier
	return dev, nil
}

func (d *Device) ConfigInterrupt() (<-chan struct{}, error) {
	interrupt, ch, err := d.setupInterrupt()
	if err != nil {
		return nil, err
	}
	kernel.StoreUint16(&d.cfg.msix_vector, interrupt)
	if res := kernel.LoadUint16(&d.cfg.msix_vector); res != interrupt {
		return nil, errors.New("virtio: failed to set up interrupt")
	}
	return ch, nil
}

func (d *Device) ConfigGeneration() uint8 {
	return kernel.LoadUint8(&d.cfg.config_generation)
}

func (d *Device) Reset() {
	// Reset device.
	kernel.StoreUint8(&d.cfg.device_status, 0)
	// Wait for reset.
	for kernel.LoadUint8(&d.cfg.device_status) != 0 {
	}
	// Acknowledge device.
	kernel.OrUint8(&d.cfg.device_status, _ACKNOWLEDGE|_DRIVER)
}

func (d *Device) Start() {
	kernel.OrUint8(&d.cfg.device_status, _DRIVER_OK)
}

// Features reads the first 64 feature bits off the device.
func (d *Device) Features() uint64 {
	// First 32 bits.
	atomic.StoreUint32(&d.cfg.device_feature_select, 0)
	feats := uint64(atomic.LoadUint32(&d.cfg.device_feature))
	// Next 32 bits.
	atomic.StoreUint32(&d.cfg.device_feature_select, 1)
	feats |= uint64(atomic.LoadUint32(&d.cfg.device_feature)) << 32
	return feats
}

func (d *Device) NegotiateFeatures(feats uint64) error {
	// First 32 bits.
	atomic.StoreUint32(&d.cfg.driver_feature_select, 0)
	atomic.StoreUint32(&d.cfg.driver_feature, uint32(feats))
	// Next 32 bits.
	atomic.StoreUint32(&d.cfg.driver_feature_select, 1)
	atomic.StoreUint32(&d.cfg.driver_feature, uint32(feats>>32))
	kernel.OrUint8(&d.cfg.device_status, _FEATURES_OK)
	if st := kernel.LoadUint8(&d.cfg.device_status); st&_FEATURES_OK == 0 {
		// Mark driver failed.
		kernel.OrUint8(&d.cfg.device_status, _FAILED)
		return errors.New("virtio: feature negotiation failed")
	}
	return nil
}

func (d *Device) ConfigureQueue(queueIndex uint16) (*Queue, error) {
	if queueIndex >= d.cfg.num_queues {
		return nil, errors.New("virtio: queue index outside range of available queues")
	}
	// Select queue.
	kernel.StoreUint16(&d.cfg.queue_select, queueIndex)
	qsz := kernel.LoadUint16(&d.cfg.queue_size)
	if qsz == 0 {
		return nil, errors.New("virtio: queue not available")
	}
	// Allocate physical memory for the queue.
	queueMemSize := int(unsafe.Sizeof(virtioDeviceQueue{}))
	mem, addr, err := allocMem(queueMemSize)
	if err != nil {
		return nil, fmt.Errorf("virtio: failed to allocate queue memory: %v", err)
	}
	if len(mem) < queueMemSize {
		return nil, fmt.Errorf("virtio: allocated %d bytes, need at least %d", len(mem), queueMemSize)
	}
	mem = mem[:queueMemSize]
	q := &Queue{
		queue:      (*virtioDeviceQueue)(unsafe.Pointer(&mem[0])),
		queueIndex: queueIndex,
	}
	// Set up queue addresses.
	descAddr := addr + unsafe.Offsetof(q.queue.descriptors)
	availAddr := addr + unsafe.Offsetof(q.queue.available)
	usedAddr := addr + unsafe.Offsetof(q.queue.used)
	atomic.StoreUint64(&d.cfg.queue_desc, uint64(descAddr))
	atomic.StoreUint64(&d.cfg.queue_driver, uint64(availAddr))
	atomic.StoreUint64(&d.cfg.queue_device, uint64(usedAddr))
	notifyAddr := d.notify.base[d.notify.multiplier*uint32(d.cfg.queue_notify_off):]
	// Ensure that the 16-bit notify fits inside the notification BAR.
	notifyAddr = notifyAddr[:2]
	q.notifyAddr = (*uint16)(unsafe.Pointer(&notifyAddr[0]))
	// Cap queue size.
	if qsz > maxQueueSize {
		qsz = maxQueueSize
		kernel.StoreUint16(&d.cfg.queue_size, qsz)
	}
	interrupt, ch, err := d.setupInterrupt()
	if err != nil {
		return nil, err
	}
	q.interrupt = ch
	kernel.StoreUint16(&d.cfg.queue_msix_vector, interrupt)
	if res := kernel.LoadUint16(&d.cfg.queue_msix_vector); res != interrupt {
		return nil, errors.New("virtio: failed to set up queue interrupt")
	}
	q.size = qsz
	// Enable queue.
	kernel.StoreUint16(&d.cfg.queue_enable, 1)
	return q, nil
}

// allocMem allocates contiguous physical memory. maxSize
// is the requested size; the returned buffer may be smaller.
func allocMem(maxSize int) ([]byte, uintptr, error) {
	pageAddr, size, err := kernel.Alloc(maxSize)
	if err != nil {
		return nil, 0, err
	}
	vmem, err := syscall.Mmap(0, 0, size, syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_ANONYMOUS)
	if err != nil {
		return nil, 0, err
	}
	vaddr := ((*reflect.SliceHeader)(unsafe.Pointer(&vmem))).Data
	if err := kernel.IOMap(vaddr, pageAddr, len(vmem)); err != nil {
		return nil, 0, err
	}
	return vmem, pageAddr, nil
}

func (d *Device) setupInterrupt() (uint16, <-chan struct{}, error) {
	if d.interrupts.table == nil {
		intTable, err := d.addr.InitInterrupts()
		if err != nil {
			return 0, nil, fmt.Errorf("virtio: no suitable interrupt support for device: %v", err)
		}
		d.interrupts.table = intTable
	}
	msixIdx := d.interrupts.usedInterrupts
	if msixIdx >= d.interrupts.table.NumInterrupts {
		return 0, nil, errors.New("virtio: no available interrupts")
	}
	notify := make(chan struct{}, 1)
	intr, err := kernel.AllocInterrupt(notify)
	if err != nil {
		return 0, nil, fmt.Errorf("virtio: failed to allocate interrupt: %v", err)
	}
	d.interrupts.usedInterrupts++
	d.interrupts.table.SetupInterrupt(msixIdx, true, intr.Addr, intr.Data)
	return uint16(msixIdx), notify, nil
}

func (d *Device) FindCapability(cap uint8) ([]byte, uint8, error) {
	if hasCaps := d.addr.ReadStatus()&(1<<3) == 0; !hasCaps {
		return nil, 0, errors.New("capacbility not found")
	}
	// Parse the linked list of capabilities.
	nextCap := d.addr.ReadCapOffset()
	for nextCap != 0 {
		capOff := nextCap
		w0 := d.addr.ReadPCIRegister(capOff)
		w1 := d.addr.ReadPCIRegister(capOff + 4)
		w2 := d.addr.ReadPCIRegister(capOff + 8)
		w3 := d.addr.ReadPCIRegister(capOff + 12)
		nextCap = uint8(w0 >> 8)
		cfgVndr := uint8(w0)
		if cfgVndr != _PCI_CAP_ID_VNDR {
			// Not a virtio capability.
			continue
		}
		if cfgTyp := uint8(w0 >> 24); cfgTyp != cap {
			continue
		}
		bar := uint8(w1)
		if bar > 0x5 {
			// Reserved BAR.
			continue
		}
		barAddr, _, isMem := d.addr.ReadBAR(bar)
		if !isMem {
			// I/O space BAR, but we only support memory mapped BARs.
			continue
		}
		offset := w2
		barAddr += uint64(offset)
		length := w3
		vmem, err := kernel.Map(uintptr(barAddr), int(length))
		if err != nil {
			return nil, 0, err
		}
		return vmem, capOff, err
	}
	return nil, 0, errors.New("capability not found")
}

type Reader struct {
	q *Queue

	// used is the ring index of the descriptor being read.
	used uint16
	// read is the number of bytes already read from the descriptor.
	read int

	// buffer is written into by the device.
	buffer []byte
	// offsets is the list of descriptor offsets into buffer.
	offsets []int
}

func NewReader(q *Queue, buffer IOMem, descSize int) (*Reader, error) {
	desc := uint16(0)
	r := &Reader{
		q:       q,
		buffer:  buffer.Mem,
		offsets: make([]int, q.size),
	}
	bufOffset := 0
loop:
	for _, b := range buffer.Blocks {
		blockOff := 0
		for {
			if desc >= q.size {
				break loop
			}
			size := descSize
			if max := b.Size - blockOff; size > max {
				size = max
			}
			if size == 0 {
				break
			}
			r.offsets[desc] = bufOffset
			q.queue.descriptors[desc] = queueDescriptor{
				addr:  uint64(b.Addr) + uint64(blockOff),
				len:   uint32(size),
				flags: _VIRTQ_DESC_F_WRITE,
			}
			bufOffset += size
			blockOff += size
			desc++
		}
	}
	if desc != q.size {
		return nil, errors.New("virtio: read buffer too small")
	}
	r.fill()
	return r, nil
}

func (r *Reader) fill() {
	counter := 0
	avail := r.q.queue.available.idx
	for {
		desc := avail
		limit := (r.used - 1)
		if (limit-desc)%r.q.size == 0 {
			break
		}
		idx := desc % r.q.size
		r.q.queue.available.ring[idx] = idx
		avail++
		counter++
	}
	r.q.notifyDevice(avail)
}

func (r *Reader) Read(buf []byte) (int, error) {
	n := 0
	var qused uint16
	for {
		qused = kernel.LoadUint16(&r.q.queue.used.idx)
		if qused != r.used {
			break
		}
		<-r.q.interrupt
	}
	for qused != r.used {
		idx := r.used % r.q.size
		desc := r.q.queue.used.ring[idx]
		if uint16(desc.id) != idx {
			return 0, errors.New("virtio: device returned descriptors out-of-order")
		}
		off := r.offsets[idx]
		src := r.buffer[off:]
		read := copy(buf, src[r.read:int(desc.len)])
		r.read += read
		buf = buf[read:]
		n += read
		if len(buf) == 0 {
			break
		}
		r.used++
		r.read = 0
	}
	r.fill()
	return n, nil
}

func (q *Queue) Size() int {
	return int(q.size)
}

type Commander struct {
	q *Queue

	// used is the used index from the last Read.
	used uint16

	freeDesc []uint16
}

func NewCommander(q *Queue) *Commander {
	c := &Commander{
		q:        q,
		freeDesc: make([]uint16, q.size, q.size),
	}
	for i := range c.freeDesc {
		c.freeDesc[i] = uint16(i)
	}
	return c
}

func (c *Commander) allocDesc() uint16 {
	idx := c.freeDesc[len(c.freeDesc)-1]
	c.freeDesc = c.freeDesc[:len(c.freeDesc)-1]
	return idx
}

func (c *Commander) Command(req, resp IOMem) bool {
	if len(c.freeDesc) < len(req.Blocks)+len(resp.Blocks) {
		used := kernel.LoadUint16(&c.q.queue.used.idx)
		if used != c.q.queue.available.idx {
			// Wait for responses.
			<-c.q.interrupt
		}
		return false
	}
	first := true
	var desc *queueDescriptor
	var idx uint16
	for _, p := range req.Blocks {
		idx = c.allocDesc()
		if desc != nil {
			desc.flags |= _VIRTQ_DESC_F_NEXT
			desc.next = idx
		}
		if first {
			c.q.queue.available.ring[c.q.queue.available.idx%c.q.size] = idx
			first = false
		}
		desc = &c.q.queue.descriptors[idx]
		*desc = queueDescriptor{
			addr: uint64(p.Addr),
			len:  uint32(p.Size),
		}
	}
	for _, p := range resp.Blocks {
		idx := c.allocDesc()
		desc.flags |= _VIRTQ_DESC_F_NEXT
		desc.next = idx
		desc = &c.q.queue.descriptors[idx]
		*desc = queueDescriptor{
			addr:  uint64(p.Addr),
			len:   uint32(p.Size),
			flags: _VIRTQ_DESC_F_WRITE,
		}
	}
	c.q.notifyDevice(c.q.queue.available.idx + 1)
	return true
}

// Read empties the device used queue and returns number of processed commands.
func (c *Commander) Read() (int, error) {
	q := &c.q.queue.used
	var count int
	for c.used != kernel.LoadUint16(&q.idx) {
		did := uint16(q.ring[c.used%c.q.size].id)
		for {
			c.freeDesc = append(c.freeDesc, did)
			desc := c.q.queue.descriptors[did]
			if desc.flags&_VIRTQ_DESC_F_NEXT == 0 {
				break
			}
			did = desc.next
		}
		c.used++
		count++
	}
	return count, nil
}

func (c *Commander) Sync() {
	for {
		used := kernel.LoadUint16(&c.q.queue.used.idx)
		if used == c.q.queue.available.idx {
			break
		}
		<-c.q.interrupt
	}
}

func (q *Queue) notifyDevice(idx uint16) {
	// Make sure that the idx increment happens after setting
	// up descriptors, and before notifying.
	kernel.StoreUint16(&q.queue.available.idx, idx)
	kernel.StoreUint16(q.notifyAddr, q.queueIndex)
}

func NewIOMem(size, capacity int) (*IOMem, error) {
	// Round up to page size.
	psize := syscall.Getpagesize()
	capacity = (capacity + psize - 1) &^ (psize - 1)
	mem, err := syscall.Mmap(0, 0, capacity, syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_ANONYMOUS)
	if err != nil {
		return nil, err
	}
	p := &IOMem{
		Mem: mem[:0],
	}
	return p, p.Ensure(size)
}

func (m *IOMem) Ensure(size int) error {
	if size > cap(m.Mem) {
		panic("buffer overflow")
	}
	for {
		need := size - len(m.Mem)
		if need <= 0 {
			break
		}
		// Allocate a chunk of contiguous physical memory and map in
		// to the end of m.mem.
		addr, got, err := kernel.Alloc(need)
		if err != nil {
			return err
		}
		vaddr := ((*reflect.SliceHeader)(unsafe.Pointer(&m.Mem))).Data
		vaddr += uintptr(len(m.Mem))
		if err := kernel.IOMap(vaddr, addr, got); err != nil {
			return fmt.Errorf("physmem: ensure: %v", err)
		}
		m.Blocks = append(m.Blocks, PhysPage{
			Addr: addr,
			Size: got,
		})
		m.Mem = m.Mem[:len(m.Mem)+got]
	}
	return nil
}

func (m *IOMem) Slice(off, end int) IOMem {
	if end > len(m.Mem) {
		panic("slicing beyond buffer end")
	}
	slice := IOMem{
		Mem: m.Mem[off:end:end],
	}
	var physOff int
	for _, p := range m.Blocks {
		addr := p.Addr
		size := p.Size
		physEnd := physOff + size
		adjust := off - physOff
		physOff = physEnd
		if adjust > 0 {
			addr += uintptr(adjust)
			size -= adjust
			if size <= 0 {
				// Block is before the range.
				continue
			}
		}
		adjust = physEnd - end
		if adjust > 0 {
			size -= adjust
			if size <= 0 {
				// Block is after the range.
				break
			}
		}
		slice.Blocks = append(slice.Blocks, PhysPage{
			Addr: addr,
			Size: size,
		})
	}
	return slice
}
