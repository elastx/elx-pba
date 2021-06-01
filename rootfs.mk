ifeq ($(ARCH),x86_64)
GOARCH := amd64
endif

rootfs-$(ARCH).img:
	GOARCH="$(GOARCH)" go run github.com/u-root/u-root -o "$(@)" \
				-initcmd pbainit \
				core \
				boot \
				github.com/elastx/elx-pba/cmd/pbainit \
				github.com/bluecmd/go-tcg-storage/cmd/sedlockctl

rootfs-$(ARCH).zst: rootfs-$(ARCH).img
	zstd -f "$(^)" -o "$@"
