#!/bin/sh

set -e

mkdir -p bootdrive/EFI/BOOT
go build -ldflags="-E eliasnaur.com/unik/kernel.rt0 -T 0x1700000" -o bootdrive/KERNEL.ELF $@
cp uefi/loader.efi bootdrive/EFI/BOOT/BOOTX64.EFI

# Create disk image
rm -f boot.img
dd if=/dev/zero of=boot.img bs=1M count=20
mformat -i boot.img ::/
mcopy -i boot.img -s bootdrive/* ::/
