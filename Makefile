PWD = $(shell pwd)
GO ?= go
DOCKER ?= docker
IP_ADDR := $(shell ipconfig getifaddr en0)

.PHONY: immugw
immugw:
	$(DOCKER) run -it --rm --name immugw -p 3324:3323 --name immugw --env IMMUGW_IMMUDB_ADDRESS=${IP_ADDR} --env IMMUGW_IMMUDB_PORT=3322 codenotary/immugw:latest

immudb:
	$(DOCKER) run --net host -it --rm --name immudb codenotary/immudb:1.3.2

.PHONY: db
db:
	printf "immudb" | ./immuadmin login immudb || true
	./immuadmin database create db1
	./immuadmin database create db2
	./immuadmin database create db3

immuadmin:
	wget https://github.com/codenotary/immudb/releases/download/v1.3.2/immuadmin-v1.3.2-darwin-arm64 .
	mv immuadmin-v1.3.2-linux-arm64 immuadmin
	chmod +x immuadmin
