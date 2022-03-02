from pathlib import Path

VERSION = "1.0.1"

def main():
    with Path("BP/hello_version.txt").open("w") as f:
        f.write(f"Hello World {VERSION}")

if __name__ == '__main__':
    main()
