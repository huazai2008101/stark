package grpc

import (
	"context"
	"fmt"

	"github.com/huazai2008101/stark"
	"github.com/huazai2008101/stark/base/log"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// 获取grpc链接
func GetGrpcConn(ctx context.Context, serviceName string) (context.Context, *grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	}
	if stark.IsEnableTrace {
		opts = append(opts,
			grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
			grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
		)
	}

	// 设置Grpc链路追踪meta信息
	ctx = setGrpcTraceMeta(ctx)

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s/%s", stark.DiscoverySchemeUrl, serviceName), opts...)
	if err != nil {
		log.Errorf(ctx, "NewGrpcConn %s 新建Grpc连接异常:%+v", serviceName, err)
	}
	return ctx, conn, err
}

// 设置Grpc链路追踪meta信息
func setGrpcTraceMeta(ctx context.Context) context.Context {
	outgoingMd, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		outgoingMd = make(metadata.MD)
	}

	incommingMd, ok := metadata.FromIncomingContext(ctx)
	if ok && len(incommingMd.Get(stark.MetadataRequestId)) > 0 {
		outgoingMd.Set(stark.MetadataRequestId, incommingMd[stark.MetadataRequestId]...)
	}

	mapCarrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, mapCarrier)
	for k, v := range mapCarrier {
		outgoingMd.Set(k, v)
	}

	if outgoingMd.Len() > 0 {
		ctx = metadata.NewOutgoingContext(ctx, outgoingMd)
	}
	return ctx
}

// 克隆上下文
func CloneContext(ctx context.Context) context.Context {
	newCtx := context.Background()

	incommingMd, ok := metadata.FromIncomingContext(ctx)
	if ok {
		newCtx = metadata.NewIncomingContext(newCtx, incommingMd)
	}

	outgoingMd, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		newCtx = metadata.NewOutgoingContext(newCtx, outgoingMd)
	}

	return newCtx
}
