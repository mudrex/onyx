install:
	go install .

build:
	env GOOS=darwin GOARCH=amd64 go build -o build/onyx-darwin-amd64 main.go
	env GOOS=linux GOARCH=amd64 go build -o build/onyx-linux-amd64 main.go
