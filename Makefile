.PHONY: build
build: client server

.PHONY: all
all: clean client server

.PHONY: client
client:
	CGO_ENABLED=0 GOARCH=amd64	GOOS=linux	go build -ldflags="$(LDFLAGS) -s -w" -o bin/aeroc_x64 ./client
	CGO_ENABLED=0 GOARCH=arm64	GOOS=linux	go build -ldflags="$(LDFLAGS) -s -w" -o bin/aeroc_arm64 ./client
	CGO_ENABLED=0 GOARCH=amd64	GOOS=windows	go build -ldflags="$(LDFLAGS) -s -w" -o bin/aeroc_x64.exe ./client

	
.PHONY: server
server:
	CGO_ENABLED=0 GOARCH=amd64	GOOS=linux	go build -ldflags="$(LDFLAGS) -s -w" -o bin/aeros_x64 ./server
	CGO_ENABLED=0 GOARCH=arm64	GOOS=linux	go build -ldflags="$(LDFLAGS) -s -w" -o bin/aeros_arm64 ./server
	CGO_ENABLED=0 GOARCH=amd64	GOOS=windows	go build -ldflags="$(LDFLAGS) -s -w" -o bin/aeros_x64.exe ./server

.PHONY: clean
clean:
	rm -f bin/*

.PHONY: compressed
compressed: build upx

.PHONY: upx
upx:
	@for f in $(shell ls bin); do upx -q -o "bin/upx_$${f}" "bin/$${f}"; done
