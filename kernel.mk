ifeq ($(ARCH),x86_64)
KERNEL_IMAGE := linux-$(LINUX_VERSION)/arch/x86_64/boot/bzImage
endif
KEYMAP ?= /usr/share/keymaps/i386/qwerty/se-latin1.kmap.gz

linux-$(LINUX_VERSION).tar.xz:
	./get-verified-tarball "$(LINUX_VERSION)" || (rm -f "$@"; exit 1)

linux-$(LINUX_VERSION)/.dir: linux-$(LINUX_VERSION).tar.xz
	tar -xf linux-$(LINUX_VERSION).tar.xz
	touch linux-$(LINUX_VERSION)/.dir

linux-$(LINUX_VERSION)/.config: linux-$(LINUX_VERSION)/.dir arch/$(ARCH)/linux.config
	cp -v "$(PWD)/arch/$(ARCH)/linux.config" "$@"
	(cd linux-$(LINUX_VERSION); make \
		ARCH="$(ARCH)" \
		olddefconfig)

linux-$(LINUX_VERSION)/drivers/tty/vt/defkeymap.c: linux-$(LINUX_VERSION)/.dir
	loadkeys -m $(KEYMAP) > linux-$(LINUX_VERSION)/drivers/tty/vt/defkeymap.c

.PHONY: linux
linux:
	make -C linux-$(LINUX_VERSION) ARCH="$(ARCH)" all -j $(shell nproc)

$(KERNEL_IMAGE)-noninteractive: linux-$(LINUX_VERSION)/.config rootfs-$(ARCH).cpio linux-$(LINUX_VERSION)/drivers/tty/vt/defkeymap.c
	make ARCH="$(ARCH)" LINUX_VERSION="$(LINUX_VERSION)" linux
	cp $(KERNEL_IMAGE) $(@)
	touch "$(@)"

$(KERNEL_IMAGE)-interactive: linux-$(LINUX_VERSION)/.config rootfs-interactive-$(ARCH).cpio linux-$(LINUX_VERSION)/drivers/tty/vt/defkeymap.c
	make ARCH="$(ARCH)" LINUX_VERSION="$(LINUX_VERSION)" CONFIG_INITRAMFS_SOURCE="../rootfs-interactive-$(ARCH).cpio" linux
	cp $(KERNEL_IMAGE) $(@)
	touch "$(@)"

