go/src/github.com/u-root/u-root/.git/HEAD:
	rm -rf go/src/github.com/u-root/u-root/ 2>/dev/null
	mkdir -p go/src/github.com/u-root/u-root/
	git clone https://github.com/u-root/u-root go/src/github.com/u-root/u-root
	(cd go/src/github.com/u-root/u-root; git reset $(UROOT_GIT_REF) --hard)

go/bin/u-root: go/src/github.com/u-root/u-root/.git/HEAD
	(cd go/src/github.com/u-root/u-root/; GOPATH=$(PWD)/go go install)
