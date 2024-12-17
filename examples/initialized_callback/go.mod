module github.com/strowk/foxy-contexts/examples/initialized_callback

go 1.23.3

replace github.com/strowk/foxy-contexts => ../../

require (
	github.com/strowk/foxy-contexts v0.0.12
	go.uber.org/fx v1.23.0
	go.uber.org/zap v1.27.0
)

require (
	go.uber.org/dig v1.18.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
)
