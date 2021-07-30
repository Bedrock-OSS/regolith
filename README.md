# regolith
Bedrock Addons compiler pipeline. 

## Running

### Install Golang

[Installation and beginners guide.](https://golang.org/doc/tutorial/getting-started)

### Install Dependencies

`go get -u ./...` to recursively install all dependencies.

### Run

Run with `go run .\main.go`

## Building as an .exe

You can build either with GoReleaser, or natively 

### Install GoReleaser

`go install github.com/goreleaser/goreleaser@latest`

### Build

`./scripts/build-local.sh`

### Or

`go build`
