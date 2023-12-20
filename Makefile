.SILENT:

build:
	go build -o bin/scatlang main.go

run: build
	./bin/scatlang hello
