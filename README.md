# Unik

Unik is a Go module for running Go programs as unikernels, without an
underlying operating system. The included demo is a functional
[Gio](https://gioui.org) GUI program that demonstrates the
[virtio](https://docs.oasis-open.org/virtio/virtio/v1.1/csprd01/virtio-v1.1-csprd01.html)
GPU and tablet drivers.

# Requirements

- Linux
- Qemu >= 4.2.0
- mtools (for creating FAT images)
- OVMF UEFI firmware (`dnf install edk2-ovmf` on Fedora)
- (Optional) gnu-efi (for building the UEFI boot loader)

# Building

The `build.sh` script takes a `go` package or file list, builds the Go
program and a bootable FAT image with the bootloader and program. To
build the demo, run

	$ ./build.sh ./cmd/demo

# Executing

The `qemu.sh` script runs the bootable image inside Qemu, with the
virtio GPU and tablet devices enabled. If everything goes well,

	$ ./qemu.sh

should give you a functional GUI program with mouse support. There is
not yet support for the keyboard input.
