ifeq ($(ARCH),x86_64)
KERNEL_IMAGE := linux-$(LINUX_VERSION)/arch/x86_64/boot/bzImage
endif

linux-$(LINUX_VERSION).tar.xz:
	./get-verified-tarball.sh "$(LINUX_VERSION)" || (rm -f "$@"; exit 1)

linux-$(LINUX_VERSION)/.dir: linux-$(LINUX_VERSION).tar.xz
	tar -xf linux-$(LINUX_VERSION).tar.xz
	touch linux-$(LINUX_VERSION)/.dir

linux-$(LINUX_VERSION)/.config: linux-$(LINUX_VERSION)/.dir arch/$(ARCH)/linux.config
	cp -v "$(PWD)/arch/$(ARCH)/linux.config" "$@"
	(cd linux-$(LINUX_VERSION); make \
		ARCH="$(ARCH)" \
		olddefconfig)

.PHONY: linux
linux:
	make -C linux-$(LINUX_VERSION) ARCH="$(ARCH)" all -j $(shell nproc)

$(KERNEL_IMAGE): linux-$(LINUX_VERSION)/.config
	make ARCH="$(ARCH)" LINUX_VERSION="$(LINUX_VERSION)" linux
	touch "$(@)"


