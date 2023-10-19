FROM docker.io/library/golang:1.19.13-bullseye

RUN apt-get update && \
	apt-get install --no-install-recommends --yes \
	gnupg2 gpgv2 flex bison build-essential libelf-dev curl \
	libssl-dev bc zstd dosfstools fdisk gdisk mtools kbd console-data libncurses-dev && \
	apt-get clean && \
	apt-get autoremove && \
	rm --force --recursive /tmp/* /var/lib/apt/lists/* /var/tmp/*

# Key IDs for torvalds@kernel.org, gregkh@kernel.org
# and autosigner@kernel.org
RUN gpg --keyserver hkps://keyserver.ubuntu.com --recv-keys \
	B8868C80BA62A1FFFAF5FDA9632D3A06589DA6B1 \
	647F28654894E3BD457199BE38DBBDC86092693E \
	ABAF11C65A2970B130ABE3C479BE3E4300411886

WORKDIR /src

CMD /usr/bin/make
