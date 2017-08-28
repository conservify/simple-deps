simple-deps: simple-deps.go
	go build -o simple-deps simple-deps.go

install: simple-deps
	cp simple-deps ~/tools/bin
