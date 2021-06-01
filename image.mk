elx-pba-$(ARCH).fs: $(KERNEL_IMAGE) rootfs-$(ARCH).zst
	truncate -s 30M "$@"
	mkfs.vfat -n ELX-PBA "$@"
	mmd -oi "$@" ::EFI
	mmd -oi "$@" ::EFI/BOOT
	mcopy -oi "$@" $< ::EFI/BOOT/linux.krn
	mcopy -oi "$@" rootfs-$(ARCH).zst ::EFI/BOOT/rootfs.zst
	mcopy -oi "$@" arch/$(ARCH)/syslinux.cfg ::EFI/BOOT/
	mcopy -oi "$@" arch/$(ARCH)/ldlinux* ::EFI/BOOT/
	mcopy -oi "$@" arch/$(ARCH)/boot*.efi ::EFI/BOOT/
	mdir -/i "$@" ::

elx-pba-$(ARCH).img: elx-pba-$(ARCH).fs
	truncate -s 32M "$@"
	sgdisk -og "$@"
	sgdisk -n "1:2048:" -c 1:"EFI System Partition" -t 1:ef00 "$@"
	dd if="$<" of="$@" seek=2048 bs=512 conv=notrunc
	sfdisk -l "$@"
