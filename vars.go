package stark

// 数据库类型
type DbType int32

const (
	DbTypeMyql  DbType = 1
	DbTypeRedis DbType = 2
)

var (
	DbTypeText = map[DbType]string{
		DbTypeMyql:  "MySQL",
		DbTypeRedis: "Redis",
	}
)

// 应用类型
type AppType int32

const (
	AppTypeWeb   AppType = 1
	AppTypeGrpc  AppType = 2
	AppTypeHttp  AppType = 3
	AppTypeCron  AppType = 4
	AppTypeQueue AppType = 5
)

var (
	AppTypeMap = map[AppType]string{
		AppTypeWeb:   "Web",
		AppTypeGrpc:  "gRPC",
		AppTypeHttp:  "Http",
		AppTypeCron:  "Cron",
		AppTypeQueue: "Queue",
	}
)

// 框架策略
type FrameworkStrategy int32

const (
	GinFrameworkStrategy  FrameworkStrategy = 1
	EchoFrameworkStrategy FrameworkStrategy = 2
)

// 服务发现策略
type DiscoveryStrategy int32

const (
	ConsulDiscoveryStrategy DiscoveryStrategy = 1
	EtcdDiscoveryStrategy   DiscoveryStrategy = 2
)

// WebInstance is *WebApplication instance May be nil
var WebInstance *WebApplication

var (
	DiscoverySchemeUrl string
	// 是否启用链路追踪功能
	IsEnableTrace bool
)
