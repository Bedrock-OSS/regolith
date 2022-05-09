<h1 align="center">Regolith</h1>

<p align="center">
<img src="https://user-images.githubusercontent.com/61835816/142045400-5b2ef154-6fb8-4055-b562-45a27bee02c0.png"></img>
</p>

<p align="center">
 • Regolith is an Add-on Compiler for the Bedrock Edition of Minecraft.
</p>
  
<p align="center">
<b>⭐ <a href="https://bedrock-oss.github.io/regolith/">Visit the Website! ⇗</a> ⭐</b>
</p>

- Much like `bridge. v2`, Regolith introduces the concept of a "project folder", where your addons are written, including the RP, the BP, and any models, textures or configuration files. This single-folder-structure is great for version control, and allows you to keep  your "source-of-truth" outside of `com.mojang`.

- In the simplest case, Regolith allows you to "compile" (copy) your RP and BP from the project folder, into the com.mojang folder. During the copy, Regolith will run a series of "filters" on the files, allowing you to programmatically transform them as they are copied into `com.mojang`. This allows you to transform your addon non-destructively, and to introduce new syntax into addons.

## 🎫 Value Preposition 

- Fundamentally, Regolith is nothing new. Many tools like this exist, such as the `bridge v2` plugin system, or even something like Gulp or SASS. 

- The value preposition for Regolith is that it allows many tools, in many languages and pay-structures to work together in a single, unified compilation flow. 

- Hobbiests can use the [Standard Library ⇗](https://github.com/Bedrock-OSS/regolith-filters), or write their own filters. Marketplace teams can write proprietary filters, or internal filters.


# 💻 Regolith Development

## 🎚 Setup:

**This setup is for developers. If you're a normal user, you can find simpler
instructions
[here](https://bedrock-oss.github.io/regolith/docs/installing).**

### 1. Install Golang

- **[Installation and beginners guide ⇗.](https://golang.org/doc/tutorial/getting-started)**

### 2. Install Dependencies

- `go get -u ./...` to recursively install all dependencies.

### 3. Test / Run

Since Regolith is a console application which requires very specific
project setup and arguments, `go run .\main.go` doesn't really do anything.

You can build and install Regolith from source using `go install` and
test test the application in the same way as user would do, by using
it on a separate Regolith project.

You can also write your own tests in the `test` directory. Here are some examples
of useful test commands:

- `go test ./test` - runs all of the tests from the test folder (we keep all of the test there)
- `go test ./test -v -run "TestRecycledCopy"` - run only the "TestRecycledCopy"
  with verbose output.
- `dlv test ./test -- "-test.run" TestInstallAllUnlockAndRun` - debug the "TestInstallAllUnlockAndRun"
  test using [delve](https://github.com/go-delve/delve)

## 🏗 Building as a `.exe`:

- You can build either with *GoReleaser*, or natively

### 1. Install GoReleaser

- `go install github.com/goreleaser/goreleaser@latest`

### 2. Build

There are a few ways to build:
 - `go install` (installs to gopath)
  - `go build` (creates a `.exe` file)
 - `./scripts/build-local.sh` (¯\_(ツ)_/¯)
