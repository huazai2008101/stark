package app

import (
	"github.com/huazai2008101/stark"
	"github.com/huazai2008101/stark/discovery/consul"
	"github.com/huazai2008101/stark/discovery/etcd"
	"github.com/huazai2008101/stark/ioc"
)

type DiscoveryAdapter struct {
	url       string
	namespace string
	strategy  stark.DiscoveryStrategy
}

func NewDiscoveryAdapter(conf *stark.DiscoveryConfig) *DiscoveryAdapter {
	if conf == nil {
		conf = &stark.DiscoveryConfig{}
	}
	return &DiscoveryAdapter{
		url:       conf.Url,
		namespace: conf.Namespace,
		strategy:  conf.Strategy,
	}
}

func (s *DiscoveryAdapter) Init() error {
	if s.url == "" {
		return nil
	}
	switch stark.DiscoveryStrategy(s.strategy) {
	case stark.ConsulDiscoveryStrategy:
		ioc.Provide(consul.NewConsulServiceDiscovery)
	default:
		ioc.Provide(etcd.NewEtcdServiceDiscovery)
	}

	s.injectProperty()
	return nil
}

func (s *DiscoveryAdapter) injectProperty() {
	// 注入配置属性
	ioc.Property("discovery.url", s.url)
	if s.namespace != "" {
		ioc.Property("discovery.namespace", s.namespace)
	}
}
