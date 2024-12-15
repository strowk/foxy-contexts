module github.com/strowk/foxy-contexts/examples/hello_world_tool

go 1.23.3

replace github.com/strowk/foxy-contexts => ../../

require (
	github.com/strowk/foxy-contexts v0.0.0-00010101000000-000000000000
	go.uber.org/fx v1.23.0
	go.uber.org/zap v1.27.0
)

require (
	go.uber.org/dig v1.18.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
