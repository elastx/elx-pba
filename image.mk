ifeq ($(ARCH),x86_64)
BOOTXEFI := bootx64.efi
endif

elx-pba-$(ARCH).fs: $(KERNEL_IMAGE)-noninteractive
	truncate -s 30M "$@"
	mkfs.vfat -n ELX-PBA "$@"
	mmd -oi "$@" ::EFI
	mmd -oi "$@" ::EFI/BOOT
	mcopy -oi "$@" $< ::EFI/BOOT/$(BOOTXEFI)
	mdir -/i "$@" ::

elx-pba-$(ARCH).img: elx-pba-$(ARCH).fs
	truncate -s 32M "$@"
	sgdisk -og "$@"
	sgdisk -n "1:2048:" -c 1:"EFI System Partition" -t 1:ef00 "$@"
	dd if="$<" of="$@" seek=2048 bs=512 conv=notrunc
	# Mark the image in the MBR region which we are not using anyway in EFI mode
	echo -n "ELX PBA IMAGE   git $(shell git rev-parse --short=12 HEAD)" | \
		dd if=/dev/stdin of="$@" count=1 bs=448 conv=notrunc
	sfdisk -l "$@"

elx-pba-interactive-$(ARCH).fs: $(KERNEL_IMAGE)-interactive
	truncate -s 30M "$@"
	mkfs.vfat -n ELX-PBA "$@"
	mmd -oi "$@" ::EFI
	mmd -oi "$@" ::EFI/BOOT
	mcopy -oi "$@" $< ::EFI/BOOT/$(BOOTXEFI)
	mdir -/i "$@" ::

elx-pba-interactive-$(ARCH).img: elx-pba-interactive-$(ARCH).fs
	truncate -s 32M "$@"
	sgdisk -og "$@"
	sgdisk -n "1:2048:" -c 1:"EFI System Partition" -t 1:ef00 "$@"
	dd if="$<" of="$@" seek=2048 bs=512 conv=notrunc
	# Mark the image in the MBR region which we are not using anyway in EFI mode
	echo -n "ELX PBA IMAGE   git $(shell git rev-parse --short=12 HEAD)" | \
		dd if=/dev/stdin of="$@" count=1 bs=448 conv=notrunc
	sfdisk -l "$@"

