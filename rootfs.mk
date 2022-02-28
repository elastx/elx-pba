ifeq ($(ARCH),x86_64)
GOARCH := amd64
endif

rootfs-$(ARCH).cpio: go/bin/u-root $(wildcard cmd/*/*.go)
	go/bin/u-root \
				-o "$(@)" \
				-build=gbb \
				-initcmd pbainit \
				core \
				github.com/u-root/u-root/cmds/exp/dmidecode \
				github.com/u-root/u-root/cmds/exp/page \
				github.com/u-root/u-root/cmds/exp/partprobe \
				github.com/elastx/elx-pba/cmd/pbainit \
				github.com/bluecmd/go-tcg-storage/cmd/sedlockctl
