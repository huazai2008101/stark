package grpc

import (
	"context"

	"github.com/huazai2008101/stark/discovery"
	"github.com/huazai2008101/stark/ioc"
	grpcModule "github.com/huazai2008101/stark/module/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/resolver"
)

type GrpcStarter struct {
	server    *grpcModule.GrpcServer     `autowire:""`
	discovery discovery.ServiceDiscovery `autowire:"?"`
}

func NewGrpcStarter() ioc.AppEvent {
	return &GrpcStarter{}
}

func (s *GrpcStarter) OnAppStart(ctx ioc.Context) {
	if s.discovery != nil {
		resolver.Register(NewBuilder(s.discovery))
	}

	grpc_health_v1.RegisterHealthServer(s.server.Server, &healthCheckServer{})
	reflection.Register(s.server.Server)
}

func (s *GrpcStarter) OnAppStop(ctx context.Context) {
	s.server.Server.GracefulStop()
}
