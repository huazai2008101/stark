package app

import (
	"github.com/huazai2008101/stark/ioc"
	grpcModule "github.com/huazai2008101/stark/module/grpc"
	grpcStarter "github.com/huazai2008101/stark/starter/grpc"
)

// 安装grpc服务
func setupGrpcServer() {
	ioc.Object(new(grpcModule.GrpcServer)).Name("grpcServer")
	ioc.Provide(grpcStarter.NewGrpcStarter).Name("grpcStarter")
}
