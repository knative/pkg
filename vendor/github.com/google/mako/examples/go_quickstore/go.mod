module github.com/google/mako-examples

go 1.12

replace github.com/google/mako => ../..

require (
	github.com/golang/protobuf v1.3.2
	github.com/google/mako v0.0.0-rc.1
	google.golang.org/grpc v1.22.1 // indirect
)
