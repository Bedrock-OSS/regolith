import sys
import time

def main():
    # Simulate some work before failing
    time.sleep(1)
    # Exit with error
    sys.exit(1)

if __name__ == '__main__':
    main()
