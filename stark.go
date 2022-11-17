package stark

import (
	"google.golang.org/grpc"
)

// 数据库连接信息
type DbConnInfo struct {
	Name string
	Url  string
	Type DbType
	// 其他配置信息
	Extras map[string]interface{}
}

// Application ...
type Application struct {
	Name        string
	Type        AppType
	Environment string
	IsDebug     bool
	SetupVars   func() error
	DbConns     []DbConnInfo
	// 服务发现配置
	Discovery *DiscoveryConfig
	// 链路追踪地址
	TraceUrl string
}

type DiscoveryConfig struct {
	// 服务发现地址
	Url string
	// 命名空间
	Namespace string
	// 服务发现策略
	Strategy DiscoveryStrategy
}

type ServerConfig struct {
	// 服务端口号
	Port int
	// 读操作超时时间
	ReadTimeout int
	// 写操作超时时间
	WriteTimeout int
	// 框架策略，1：gin，2：echo
	Strategy FrameworkStrategy
	// 是否启用swagger
	EnableSwagger bool
}

// WebApplication ...
type WebApplication struct {
	*Application
	*ServerConfig
	GrpcServerOptions []grpc.ServerOption
}
