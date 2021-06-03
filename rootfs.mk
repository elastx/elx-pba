ifeq ($(ARCH),x86_64)
GOARCH := amd64
endif

rootfs-$(ARCH).cpio: $(wildcard cmd/*/*.go)
	GOARCH="$(GOARCH)" go run github.com/u-root/u-root -o "$(@)" \
				-initcmd pbainit \
				core \
				boot \
				github.com/u-root/u-root/cmds/exp/dmidecode \
				github.com/u-root/u-root/cmds/exp/page \
				github.com/elastx/elx-pba/cmd/pbainit \
				github.com/bluecmd/go-tcg-storage/cmd/sedlockctl
