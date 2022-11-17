package gin

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// 注册swagger路由
type SwaggerRouter struct {
	group *gin.RouterGroup `autowire:""`
}

func (s *SwaggerRouter) Init() {
	s.group.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
