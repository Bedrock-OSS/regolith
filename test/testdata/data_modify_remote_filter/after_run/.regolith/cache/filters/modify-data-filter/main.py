'''
This is a copy of the run_counter filter:
https://github.com/Nusiq/regolith-filters/tree/master/run_counter

It adds a number to the output file in the data directory. It's used for
testing opt-in data export filters on the Regolith project.
'''
from pathlib import Path

COUNTER_PATH = Path('data/modify-data-filter/counter.txt')


def main():
    if not COUNTER_PATH.exists():
        COUNTER_PATH.parent.mkdir(parents=True, exist_ok=True)
        COUNTER_PATH.write_text('0')

    counter = int(COUNTER_PATH.read_text())
    counter += 1
    COUNTER_PATH.write_text(str(counter))


if __name__ == "__main__":
    main()
