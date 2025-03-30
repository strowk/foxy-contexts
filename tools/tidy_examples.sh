#!/bin/bash

set -e

( cd examples/git_repository_resource && go mod tidy )
( cd examples/hello_world_resource && go mod tidy )
( cd examples/k8s_contexts_resources && go mod tidy )
( cd examples/list_current_dir_files_tool && go mod tidy )
( cd examples/list_k8s_contexts_tool && go mod tidy )
( cd examples/streamable_http && go mod tidy )
( cd tests/lifecycle && go mod tidy )

