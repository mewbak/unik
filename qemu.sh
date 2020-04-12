#!/bin/sh

set -e

qemu-system-x86_64 -enable-kvm -machine q35 -net none -drive if=pflash,format=raw,readonly,file=/usr/share/OVMF/OVMF_CODE.fd -drive media=cdrom,format=raw,readonly,file=boot.img -vga virtio -display sdl,gl=on -device virtio-tablet-pci -serial stdio $@
