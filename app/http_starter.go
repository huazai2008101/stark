package app

import (
	"context"
	"net"
	"net/http"

	"github.com/huazai2008101/stark"
	"github.com/huazai2008101/stark/base/log"
	"github.com/huazai2008101/stark/discovery"
	"github.com/huazai2008101/stark/ioc"
)

type HttpStarter struct {
	name      string                     `value:"${application.name}"`
	port      int                        `value:"${application.port}"`
	listener  *net.TCPListener           `autowire:""`
	server    *http.Server               `autowire:""`
	mux       *ServeMux                  `autowire:""`
	discovery discovery.ServiceDiscovery `autowire:"?"`
}

func NewHttpStarter() ioc.AppEvent {
	return &HttpStarter{}
}

func (s *HttpStarter) OnAppStart(ctx ioc.Context) {
	s.mux.Init()
	s.server.Handler = s.mux
	log.Infof(ctx.Context(), "%s 正在启动服务 端口号:%d", s.name, s.port)
	ioc.Go(func(ctx context.Context) {
		s.server.Serve(s.listener)
	})

	// 如果有服务发现机制则进行注册服务
	if s.discovery != nil {
		stark.DiscoverySchemeUrl = s.discovery.SchemeUrl()
		err := s.discovery.Register()
		if err != nil {
			log.Errorf(ctx.Context(), "%s 注册服务异常:%+v", s.name, err)
			panic(err)
		}
	}
}

func (s *HttpStarter) OnAppStop(ctx context.Context) {
	s.server.Close()
	s.listener.Close()
}
