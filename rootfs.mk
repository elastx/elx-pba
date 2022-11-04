ifeq ($(ARCH),x86_64)
GOARCH := amd64
endif

rootfs-$(ARCH).cpio: go/bin/u-root $(wildcard cmd/*/*.go)
	(cd go/src/github.com/u-root/u-root; ../../../../bin/u-root \
				-o "../../../../../$(@)" \
				-build=gbb \
				-initcmd pbainit \
				boot \
				core \
				./cmds/exp/dmidecode \
				./cmds/exp/page \
				./cmds/exp/partprobe \
				../../../../../cmd/pbainit \
				../../open-source-firmware/go-tcg-storage/cmd/sedlockctl \
				../../open-source-firmware/go-tcg-storage/cmd/tcgdiskstat \
				../../open-source-firmware/go-tcg-storage/cmd/tcgsdiag \
	)

rootfs-interactive-$(ARCH).cpio: go/bin/u-root $(wildcard cmd/*/*.go)
	(cd go/src/github.com/u-root/u-root; ../../../../bin/u-root \
				-o "../../../../../$(@)" \
				-build=gbb \
				-initcmd pbainit-interactive \
				boot \
				core \
				./cmds/exp/dmidecode \
				./cmds/exp/page \
				./cmds/exp/partprobe \
				../../../../../cmd/pbainit-interactive \
				../../open-source-firmware/go-tcg-storage/cmd/sedlockctl \
				../../open-source-firmware/go-tcg-storage/cmd/tcgdiskstat \
				../../open-source-firmware/go-tcg-storage/cmd/tcgsdiag \
	)
