package grpc

import "google.golang.org/grpc/resolver"

type consulResolver struct {
	cc           resolver.ClientConn
	serviceName  string
	discoveryUrl string
}

func newConsulResolver(cc resolver.ClientConn, scheme, discoveryUrl, serviceName string) (*consulResolver, error) {
	return &consulResolver{
		cc:           cc,
		serviceName:  serviceName,
		discoveryUrl: discoveryUrl,
	}, nil
}

func (c *consulResolver) ResolveNow(o resolver.ResolveNowOptions) {}

func (c *consulResolver) Close() {}
