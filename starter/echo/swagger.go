package echo

import (
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// 注册swagger路由
type SwaggerRouter struct {
	group *echo.Group `autowire:""`
}

func (s *SwaggerRouter) Init() {
	s.group.GET("/swagger/*any", echoSwagger.WrapHandler)
}
