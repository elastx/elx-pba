go/src/github.com/u-root/u-root/.git/HEAD:
	mkdir -p go/src/github.com/u-root/u-root/
	git clone https://github.com/u-root/u-root go/src/github.com/u-root/u-root

go/bin/u-root: go/src/github.com/u-root/u-root/.git/HEAD
	(cd go/src/github.com/u-root/u-root/; GOPATH=$(PWD)/go go install)
