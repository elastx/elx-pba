# elx-pba

Pre-boot authentication image for TCG Storage devices

## Building

**NOTE**: Due to u-root not supporting 1.16 yet, you have to use a Go version
lower than 1.16 to build - e.g. 1.15.

```
$ sudo apt install \
    gnupg2 gpgv2 flex bison build-essential libelf-dev \
    curl libssl-dev bc zstd dosfstools gdisk mtools
$ gpg2 --locate-keys torvalds@kernel.org gregkh@kernel.org autosigner@kernel.org
# Make sure sgdisk is in the PATH
$ PATH=$PATH:/sbin make
```

## Testing

```
$ OPAL_KEY=debug
$ sudo sedutil-cli --loadpbaimage "${OPAL_KEY}" elx-pba-x86_64.img /dev/sdb
```
