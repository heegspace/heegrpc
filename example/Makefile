export GOPROXY=https://goproxy.io
default: build

build: export GO111MODULE=on

build:
	rm -rf bin
	

	go build -o service service.go
	go build -o client client.go

	mkdir bin
	cp -rf service bin/
	cp -rf client bin/

clean:
	rm -rf bin
	rm client
	rm service
