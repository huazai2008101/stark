package grpc

import "google.golang.org/grpc"

// 服务端可选参数
type ServerOptions struct {
	Options []grpc.ServerOption
}

// 可选参数
type ClientOptions struct {
	Options []grpc.DialOption
}
