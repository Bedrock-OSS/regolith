# regolith

Regolith is an Addon Compiler for the Bedrock Edition of Minecraft. 

⭐ [Visit the website!](https://bedrock-oss.github.io/regolith/)⭐

Much like `bridge v2`, Regolith introduces the concept of a "project folder", where your addons are written, including the RP, the BP, and any models, textures or configuration files. This single-folder-structure is great for version control, and allows you to keep  your "source-of-truth" outside of `com.mojang`.

In the simplest case, Regolith allows you to "compile" (copy) your RP and BP from the project folder, into the com.mojang folder. During the copy, Regolith will run a series of "filters" on the files, allowing you to programmatically transform them as they are copied into `com.mojang`. This allows you to transform your addon non-destructively, and to introduce new syntax into addons.

## Value Preposition 

Fundamentally, Regolith is nothing new. Many tools like this exist, such as the `bridge v2` plugin system, or even something like Gulp or SASS. 

The value preposition for Regolith is that it allows many tools, in many languages and pay-structures to work together in a single, unified compilation flow. 

Hobbiests can use the Standard Library, or write their own filters. Marketplace teams can write proprietary filters, or internal filters.


# Regolith Development

## Setup

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

### Or

`go install`
