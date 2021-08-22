# regolith

Regolith is an Addon Compiler for the Bedrock Edition of Minecraft. 

Much like `bridge v2`, Regolith introduces the concept of a "project folder", where your addons are written, including the RP, the BP, and any models, textures or configuration files. This single-folder-structure is great for version control, and allows you to keep  your "source-of-truth" outside of `com.mojang`.

In the simplest case, Regolith allows you to "compile" (copy) your RP and BP from the project folder, into the com.mojang folder. During the copy, Regolith will run a series of "filters" on the files, allowing you to programmatically transform them as they are copied into `com.mojang`. This allows you to transform your addon non-destructively, and to introduce new syntax into addons.

## Value Preposition 

Fundamentally, Regolith is nothing new. Many tools like this exist, such as the `bridge v2` plugin system, or even something like Gulp or SASS. 

The value preposition for Regolith is that it allows many tools, in many languages and pay-structures to work together in a single, unified compilation flow. 

Hobbiests can use the Standard Library, or write their own filters. Marketplace teams can write proprietary filters, or internal filters.

## Compilation Flow

The compilation flow in Regolith is like this:
 - Copy RP and BP into a `temp` folder.
 - Run each filter in order, allowing it to destructively edit `temp`
 - Copy RP and BP from `temp` into `com.mojang`, or another export target

## Filters

A `filter` is any process that edits/adds/deletes files inside of the `temp` folder. Each filter is allowed to change this folder, before passing the results along to the next filter. That allows filters to build on each other.

Examples of potential filters:
 - A filter that removes all comments from json, allowing the next filter(s) to read json without needing to handle comments
 - A filter which automatically generates `texture_list.json`
 - A filter which takes a `.psd` file, and automatically exports the png, allowing you to work directly in photoshop, without needing to export the files each time you edit it.
 - A filter which generates `1.10` format items, based on item-textures in a specific folder.
 - A filter which automatically inserts your team-branding maps or entities.

### Local Filters

Local filters are files that you write yourself, and place into the regolith project. These can be written in Java, Python, or Node JS. You can run register these files as a filter, and they will be run during compilation.

### Online Filters

You can also reference filters from github, by simply adding the github link as a filter. The relevant files will be downloaded into the regolith cache, and then run.

### Standard Library

The Regolith Standard Library is a git repository which contains "official" filters. This is the safest way to use Regolith, and the least technical.

Source for all standard library filters can be found [here](https://github.com/Bedrock-OSS/regolith-filters)

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
