---
title: Experiments
---

# Experiments

Experiments are new experimental features of Regolith to be released in the future versions, once proven to be stable and useful. The experiments can be enabled with the `--experiments` flag.

## Currently Available Experiments

### `size_time_check`

The `size_time_check` is an experiment that aims to speed up `regolith run` and `regolith watch` commands. It achieves this by checking the size and modification time of the files before moving them between working and output directories. If the source file is the same size and has the same modification time as the destination file, the target file will remain untouched (Regolith assumes that the files are the same).

The `size_time_check` should greatly speed up the exports of large projects.

The downside of this approach is that on the first run, the export will be slower, but on subsequent runs, the export will be much faster. This means that the `size_time_check` is not recommended for CI where Regolith is run only once.

Usage:
```
regolith run --experiments size_time_check
regolith watch --experiments size_time_check
```