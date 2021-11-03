This folder is used to store resources used for testing.

- `fresh_project` - the regolith project created with `regolith init` and
    nothing else. It uses files called *.ignoreme* to simulate empty paths
    which aren't supported by git.
- `miniaml_project` - the simplest possible valid project, no filters but with
    addition of *manifest.json* for BP and RP, and with empty file in data
    path.
- `multitarget_project` - a copy of `minimal_project` but with modified
    config.json, to add multiple profiles with different export targets.
