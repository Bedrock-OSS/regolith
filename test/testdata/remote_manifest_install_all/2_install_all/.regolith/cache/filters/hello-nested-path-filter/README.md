> [!WARNING]
> This is the `hello-nested-path-filter` filter. The folder is named differently than the filter on purpose, to test if the path can be different.

This filter creates a `BP/hello_nested_path_filter.txt` file and and writes its version and the the content of `data/hello-nested-path/message.txt` into it.

The default message is "Hello from hello-nested-path-filter!" (and it's stored in the filter's default data in the message.txt file).

The output should look like:
```
Version: 1.0.0
Message: Hello from hello-nested-path-filter!
```
