module github.com/strowk/foxy-contexts/examples/simple_prompt

go 1.23.3

replace github.com/strowk/foxy-contexts => ../../

require (
	github.com/strowk/foxy-contexts v0.0.0-00010101000000-000000000000
	go.uber.org/fx v1.23.0
	go.uber.org/zap v1.26.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	go.uber.org/dig v1.18.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
)
