# Contributing

## Overview

I welcome any contributions to this project. Please follow the guidelines below.

## Issues

If you find a bug or have a feature request, please open an issue in Github. If you are able to provide test to reproduce the issue, that would be very helpful.

## Coding

This project uses Go modules, so you should be able to clone the repository to any location on your machine.

## Testing

`go test ./...` will run library tests.

Following could run tests in examples:

```bash
cd examples/git_repository_resource && go test ./... ; cd -
cd examples/hello_world_resource && go test ; cd -
cd examples/k8s_contexts_resources && go test ; cd -
cd examples/list_current_dir_files_tool && go test ; cd -
cd examples/list_k8s_contexts_tool && go test ; cd -
cd examples/streamable_http && go test ; cd -
cd examples/resource_provider && go test ; cd -
cd tests/lifecycle && go test ; cd -
```

## Documentation

Documentation is written in "docs" folder and is built using Hugo.

See how install it here: https://gohugo.io/installation/

To develop documentation, run:

```bash
hugo server
```

First time it would need to download a bunch of stuff.
When that is done, open `http://localhost:1313/` in your browser to see current docs.
Every change you save would be automatically reflected in browser.


