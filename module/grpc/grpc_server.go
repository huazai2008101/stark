package grpc

import (
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/huazai2008101/stark/ioc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

type GrpcServer struct {
	Server   *grpc.Server
	options  *ServerOptions `autowire:"?"`
	traceUrl string         `value:"${trace.url:=}"`
}

func (s *GrpcServer) OnInit(ctx ioc.Context) error {
	var opts []grpc.ServerOption
	if s.options != nil {
		opts = append(opts, s.options.Options...)
	}

	// 如果链路追踪地址存在则配置链路追踪拦截
	if s.traceUrl == "" {
		opts = append(opts, grpc.UnaryInterceptor(grpcMiddleware.ChainUnaryServer(
			grpcRecovery.UnaryServerInterceptor(),
		)))
	} else {
		opts = append(opts, grpc.UnaryInterceptor(grpcMiddleware.ChainUnaryServer(
			grpcRecovery.UnaryServerInterceptor(),
			otelgrpc.UnaryServerInterceptor(),
		)))
		opts = append(opts, grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()))
	}

	s.Server = grpc.NewServer(opts...)
	return nil
}
