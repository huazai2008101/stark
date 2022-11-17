package app

import (
	"github.com/gin-gonic/gin"
	"github.com/huazai2008101/stark"
	"github.com/huazai2008101/stark/ioc"
	"github.com/huazai2008101/stark/module/web"
	echoStarter "github.com/huazai2008101/stark/starter/echo"
	ginStarter "github.com/huazai2008101/stark/starter/gin"
	"github.com/labstack/echo/v4"
)

type WebFrameworkAdapter struct {
	strategy      stark.FrameworkStrategy
	enableSwagger bool
}

func NewWebFrameworkAdapter(conf *stark.ServerConfig) *WebFrameworkAdapter {
	if conf == nil {
		conf = &stark.ServerConfig{
			Strategy: stark.GinFrameworkStrategy,
		}
	}
	return &WebFrameworkAdapter{
		strategy:      conf.Strategy,
		enableSwagger: conf.EnableSwagger,
	}
}

func (s *WebFrameworkAdapter) Init() error {
	switch stark.FrameworkStrategy(s.strategy) {
	case stark.EchoFrameworkStrategy:
		s.initEcho()
	default:
		s.initGin()
	}
	return nil
}

func (s *WebFrameworkAdapter) initGin() {
	engine := gin.New()
	ioc.Object(engine)
	ioc.OnProperty("application.name", func(val string) {
		ioc.Object(engine.Group(val))
	})
	ioc.Provide(ginStarter.NewGinStarter).Name("ginStarter")

	// 判断是否启用swagger文档功能
	if s.enableSwagger {
		ioc.Object(new(ginStarter.SwaggerRouter)).Export((*web.RouteInitializer)(nil))
	}
}

func (s *WebFrameworkAdapter) initEcho() {
	engine := echo.New()
	ioc.Object(engine)
	ioc.OnProperty("application.name", func(val string) {
		ioc.Object(engine.Group(val))
	})
	ioc.Provide(echoStarter.NewEchoStarter).Name("echoStarter")

	// 判断是否启用swagger文档功能
	if s.enableSwagger {
		ioc.Object(new(echoStarter.SwaggerRouter)).Export((*web.RouteInitializer)(nil))
	}
}
