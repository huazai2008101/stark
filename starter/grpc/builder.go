package grpc

import (
	"context"
	"fmt"
	"path"

	"github.com/huazai2008101/stark/base/log"
	"github.com/huazai2008101/stark/discovery"
	"google.golang.org/grpc/resolver"
)

type resolverBuilder struct {
	discovery discovery.ServiceDiscovery
}

func NewBuilder(discovery discovery.ServiceDiscovery) resolver.Builder {
	return &resolverBuilder{
		discovery: discovery,
	}
}

func (s *resolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	ctx := context.Background()

	r, err := newConsulResolver(cc, target.URL.Scheme, target.URL.Host, target.URL.Path)
	if err != nil {
		return nil, err
	}

	serviceName := path.Base(target.URL.Path)
	instanceList := s.discovery.ServiceInstances(serviceName)
	if len(instanceList) == 0 {
		log.Errorf(ctx, "resolverBuilder 没有可用服务实例 %s", serviceName)
		return nil, fmt.Errorf("没有可用服务实例")
	}
	resolverState := resolver.State{}
	for _, v := range instanceList {
		resolverState.Addresses = append(resolverState.Addresses, resolver.Address{
			Addr:       fmt.Sprintf("%s:%d", v.Address, v.Port),
			ServerName: v.Name,
		})
	}
	err = cc.UpdateState(resolverState)
	if err != nil {
		log.Errorf(ctx, "resolverBuilder 更新grpc连接状态异常:%+v", err)
		return nil, err
	}

	return r, nil
}

func (s *resolverBuilder) Scheme() string {
	return s.discovery.SchemeName()
}
