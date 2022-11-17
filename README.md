

# stark

stark是新瑞鹏基础框架，基于IOC思想实现

## 快速开始

1. 安装依赖包

   ```go
   go get github.com/huazai2008101/stark
   ```

2. 添加main文件入口

   ```go
   package main
   
   import (
   	"flag"
   
   	_ "stark-web-demo/docs"
   	_ "stark-web-demo/grpc"
   	_ "stark-web-demo/http"
   
   	"github.com/limitedlee/microservice/common/config"
   	"github.com/maybgit/glog"
   	"github.com/huazai2008101/stark"
   	"github.com/huazai2008101/stark/app"
   	"google.golang.org/grpc"
   )
   
   // @title 统一登录项目接口文档
   // @version 1.0
   // @description 这里是描述
   // @host 10.1.1.248:11080
   func main() {
   	flag.Parse()
   	defer glog.Flush()
   
   	instance := &stark.WebApplication{
   		Application: &stark.Application{
   			Name:        "test",
   			Type:        stark.AppTypeWeb,
   			Environment: "",
   			SetupVars: func() error {
   				return nil
   			},
   			DbConns: []stark.DbConnInfo{
   				{
   					Name: "stock",
   					Url:  config.GetString("mysql.dc_oms"),
   					Type: stark.DbTypeMyql,
   				},
   				{
   					Name: "redis",
   					Url:  config.GetString("redis.Addr"),
   					Type: stark.DbTypeRedis,
   					Extras: map[string]interface{}{
   						"username": config.GetString("redis.Password"),
   					},
   				},
   			},
   			Discovery: &stark.DiscoveryConfig{
   				Url:       "127.0.0.1:8500",
   				Namespace: "",
   				Strategy:  stark.ConsulDiscoveryStrategy,
   			},
   		},
   		ServerConfig: &stark.ServerConfig{
   			Port:          9000,
   			Strategy:      stark.GinFrameworkStrategy,
   			EnableSwagger: true,
   		},
   		GrpcServerOptions: []grpc.ServerOption{
   			grpc.MaxRecvMsgSize(1024 * 1024),
   			grpc.MaxSendMsgSize(1024 * 1024),
   		},
   	}
   	app.RunWebApplication(instance)
   }
   ```

   参考demo地址：[Stark Web Demo ](https://github.com/huazai2008101/stark-web-demo)

3. 注册http服务路由

   需要实现以下接口

   ```go
   package web
   
   // 路由初始化接口
   type RouteInitializer interface {
   	Init()
   }
   ```

   

   ```go
   package http
   
   import (
   	"context"
   	"net/http"
   	"stark-web-demo/dto"
   	"stark-web-demo/service"
   	"stark-web-demo/util"
   
   	"github.com/gin-gonic/gin"
   	"github.com/huazai2008101/stark/base/log"
   )
   
   type warehouseHttpServer struct {
   	warehouseService *service.WarehouseService `autowire:""`
   	router           *gin.RouterGroup          `autowire:""`
   }
   
   // @Summary 查询类型列表
   // @Tags 授权管理
   // @Accept json
   // @Produce json
   // @Param page_index query int true "分页页码"
   // @Param page_size query int true "分页大小"
   // @Success 200 {object}  dto.CommonHttpResponse{data=dto.WarehouseCategory{}}
   // @Router /auth-center/auth-manage [GET]
   func (s *warehouseHttpServer) ListCategory(ctx *gin.Context) {
   	resp := dto.CommonHttpResponse{
   		Code:    http.StatusBadRequest,
   		Message: "",
   		Data:    nil,
   	}
   	categoryList, err := s.warehouseService.ListCategory(context.Background(), dto.QueryWarehouseCategoryParam{})
   	if err != nil {
   		log.Errorf("warehouseHttpServer/ListCategory 查询类型列表异常:%+v", err)
   		resp.Message = err.Error()
   		util.ResponseJSON(ctx, resp)
   		return
   	}
   
   	list := make([]dto.WarehouseCategory, 0, len(categoryList))
   	for _, v := range categoryList {
   		list = append(list, dto.WarehouseCategory{
   			Id:   v.Id,
   			Code: v.Code,
   			Name: v.Name,
   		})
   	}
   
   	resp.Code = http.StatusOK
   	resp.Data = list
   	util.ResponseJSON(ctx, resp)
   }
   
   // 注册路由
   func (s *warehouseHttpServer) Init() {
   	group := s.router.Group("/warehouse")
   	group.GET("/category", s.ListCategory)
   }
   ```

   ```go
   package http
   
   import (
   	_ "stark-web-demo/service"
   
   	"github.com/huazai2008101/stark/ioc"
   	"github.com/huazai2008101/stark/module/web"
   )
   
   func init() {
   	ioc.Provide(new(warehouseHttpServer)).Export((*web.RouteInitializer)(nil))
   }
   ```

4. 注册grpc服务

   需要实现以下接口，在OnInit注册grpc服务

   ```go
   type BeanInit interface {
   	OnInit(ctx Context) error
   }
   ```

   ```go
   package service
   
   import (
   	"context"
   	"stark-web-demo/service"
   
   	"stark-web-demo/proto/helloworld"
   
   	"github.com/huazai2008101/stark/ioc"
   	"github.com/huazai2008101/stark/module/grpc"
   )
   
   type helloGrpcServer struct {
   	server           *grpc.GrpcServer          `autowire:""`
   	warehouseService *service.WarehouseService `autowire:""`
   }
   
   func (s *helloGrpcServer) OnInit(ctx ioc.Context) error {
   	helloworld.RegisterGreeterServer(s.server.Server, s)
   	return nil
   }
   
   func (s *helloGrpcServer) SayHello(context.Context, *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
   	return &helloworld.HelloReply{
   		Message: "hello world",
   	}, nil
   }
   ```

   ```go
   package service
   
   import (
   	_ "stark-web-demo/service"
   
   	"github.com/huazai2008101/stark/ioc"
   )
   
   func init() {
   	ioc.Object(new(helloGrpcServer))
   }
   ```

   

3. 注意事项：

   a.项目包有分层关系的，在main.go文件需要引入【_ "stark-web-demo/grpc"】和【_ "stark-web-demo/http"】包，而在grpc和http层需要引入【_ "stark-web-demo/cache"】和【_ "stark-web-demo/repository"】包

   b.如果启用swagger文档功能则main.go文件需要引入【_ "stark-web-demo/docs"】，否则会报请求doc.json报错

   

## IOC使用指南

1. 注册对象，注意以下操作必须在ioc容器bean刷新完成前调用，否则会报错

   ```go
   # 通过Object方法注册bean
   ioc.Object(new(WarehouseCache))
   
   # 给对象自定义bean名称
   ioc.Object(new(WarehouseCache)).Name("myWarehouseCache")
   
   # 给对象指定优先级顺序
   ioc.Object(new(WarehouseCache)).Order(1000)
   
   # 指定bean导出接口类型
   ioc.Object(new(WarehouseCache)).Export((ifc.WarehouseInterface)(nil))
   
   # bean注册成功初始化回调
   ioc.Object(new(warehouseHttpServer)).Init(func(c *warehouseHttpServer) {
       c.route()
   })
   ```

   ```go
   # 通过Provide构造方法注册bean
   // func NewWarehouseHttpServer(router *gin.RouterGroup) *warehouseHttpServer {
   // 	return &warehouseHttpServer{
   // 		router: router,
   // 	}
   // }
   ioc.Provide(NewWarehouseHttpServer,(*gin.RouterGroup)(nil))
   ```

2. 注册配置

   ```go
   ioc.Property("consul.url", consulUrl)
   ```

3. 注入对象

   根据对象类型注入

   ```go
   type WarehouseCache struct {
   	redisClient *redis.Client `autowire:""`
   }
   ```

   根据bean名称注入

   ```go
   type WarehouseService struct {
   	stockDb             *gorm.DB                       `autowire:"stock"`
   }
   ```

   单个bean可选注入

   ```go
   type HttpStarter struct {
   	discovery discovery.ServiceDiscovery `autowire:"?"`
   }
   ```

   多个bean注入

   ```go
   type GinStarter struct {
   	// 用户自定义中间件
   	middlewares []gin.HandlerFunc `autowire:""`
   }
   ```

   多个bean可选注入

   ```go
   type GinStarter struct {
   	// 用户自定义中间件
   	middlewares []gin.HandlerFunc `autowire:"*?"`
   }
   ```

4. 注入配置

   ```
   type GinStarter struct {
   	name        string           `value:"${application.name}"` //无默认值
   	// 链路追踪服务器地址
   	jaegerUrl string `value:"${jaeger.url:=http://127.0.0.1:8320}"` // 有默认值，默认值为“http://127.0.0.1:8320”
   	// 不记录日志的路由
   	excludeLogPaths []string `value:"${application.log.excludePath:=}"`
   }
   ```
	监听配置
   ```go
   ioc.OnProperty("application.name", func(val string) {
       ioc.Object(engine.Group(val))
   })
   ```


## 启用swagger
1. main.go添加对应swagger文档注释

2. 在main.go文件对应路径下，通过执行以下命令生成swagger接口文档

   ```shell
   swag init
   ```

3. ServerConfig.EnableSwagger值设置为true，然后在main.go文件导入swagger包名(_ "{项目包名}/docs")

   ```go
   package main
   
   import (
   	"flag"
   
   	_ "stark-web-demo/docs"
   	_ "stark-web-demo/grpc"
   	_ "stark-web-demo/http"
   
   	"github.com/limitedlee/microservice/common/config"
   	"github.com/maybgit/glog"
   	"github.com/huazai2008101/stark"
   	"github.com/huazai2008101/stark/app"
   	"google.golang.org/grpc"
   )
   
   // @title 统一登录项目接口文档
   // @version 1.0
   // @description 这里是描述
   // @host 10.1.1.248:11080
   func main() {
   	flag.Parse()
   	defer glog.Flush()
   
   	instance := &stark.WebApplication{
   		Application: &stark.Application{
   			Name:        "test",
   			Type:        stark.AppTypeWeb,
   			Environment: "",
   			SetupVars: func() error {
   				return nil
   			},
   			DbConns: []stark.DbConnInfo{
   				{
   					Name: "stock",
   					Url:  config.GetString("mysql.dc_oms"),
   					Type: stark.DbTypeMyql,
   				},
   				{
   					Name: "redis",
   					Url:  config.GetString("redis.Addr"),
   					Type: stark.DbTypeRedis,
   					Extras: map[string]interface{}{
   						"username": config.GetString("redis.Password"),
   					},
   				},
   			},
   			Discovery: &stark.DiscoveryConfig{
   				Url:       "127.0.0.1:8500",
   				Namespace: "",
   				Strategy:  stark.ConsulDiscoveryStrategy,
   			},
   		},
   		ServerConfig: &stark.ServerConfig{
   			Port:          9000,
   			Strategy:      stark.GinFrameworkStrategy,
   			EnableSwagger: true,
   		},
   		GrpcServerOptions: []grpc.ServerOption{
   			grpc.MaxRecvMsgSize(1024 * 1024),
   			grpc.MaxSendMsgSize(1024 * 1024),
   		},
   	}
   	app.RunWebApplication(instance)
   }
   ```

## grpc相关用法
实例化grpc连接，ctx中存在“x-request-id”则会传递给下一个grpc请求

```go
conn,err:=app.NewGrpcConn(ctx,"test")
```

从gin context中实例化grpc连接，从gin请求头提取“x-request-id"并放入ctx metadata里面

```go
// ctx *gin.Context
conn, err := app.NewGinGrpcConn(ctx, "test")
```

从echo context中实例化grpc连接，从gin请求头提取“x-request-id"并放入ctx metadata里面

```go
// ctx echo.Context
conn, err := app.NewEchoGrpcConn(ctx, "test")
```



## 链路日志打印

先从上下文中获取日志对象

```go
// log包不要引用错误，是github.com/huazai2008101/stark/base/log
log:=log.WithContext(ctx)
```

再调用日志相关打印方法

```go
log.Errorf("warehouseHttpServer/ListCategory 查询类型列表异常:%+v", err)
log.Infof("warehouseHttpServer/ListCategory 查询类型列表异常:%+v", err)
```

