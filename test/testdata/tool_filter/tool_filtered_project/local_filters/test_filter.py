import sys

def main():
    name = sys.argv[1]
    with open('BP/hello.txt', 'w') as f:
        f.write(f'Hello {name}!')

if __name__ == "__main__":
    main()
