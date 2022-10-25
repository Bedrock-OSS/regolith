'''
Simple testing regolith filter which prints to out.txt file of BP
The text being printed is configured in the filter's config in config.json
file.
'''
import sys
import json
from pathlib import Path

BP_PATH = Path('BP')

def main():
    config = json.loads(sys.argv[1])
    output_text = config['output_text']
    (BP_PATH / 'out.txt').write_text(output_text, encoding='utf8')

if __name__ == "__main__":
    main()