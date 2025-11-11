import sys
import time
import json

def main():
    data = json.loads(sys.argv[1])
    wait_time, output, message = data['wait_time'], data['output'], data['message']
    time.sleep(int(wait_time))
    with open(output, 'w') as file:
        file.write(message)

if __name__ == '__main__':
    main()
