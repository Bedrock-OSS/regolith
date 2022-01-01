This folder is used to store resources used for testing.

- `fresh_project` - the regolith project created with `regolith init` and
    nothing else. It uses files called *.ignoreme* to simulate empty paths
    which aren't supported by git.
- `miniaml_project` - the simplest possible valid project, no filters but with
    addition of *manifest.json* for BP and RP, and with empty file in data
    path.
- `multitarget_project` - a copy of `minimal_project` but with modified
    config.json, to add multiple profiles with different export targets.
- `double_remote_project` - a project that uses a remote filter from
    [regolith-test-filters](https://github.com/Bedrock-OSS/regolith-test-filters).
    The filter has a reference to another remote filter on the same reposiotry.
- `double_remote_project_installed` - expected result of contents of
    `double_remote_project` after installation.
- `run_missing_rp_project` - a project that for testing `regolith run` which is
  missing `packs/RP`. The profile doesn't have any filters.
