package app

import (
	"context"

	"github.com/huazai2008101/stark"
	"github.com/huazai2008101/stark/base/log"
	"github.com/huazai2008101/stark/ioc"
	grpcModule "github.com/huazai2008101/stark/module/grpc"
)

// RunWebApplication runs http and grpc application.
func RunWebApplication(application *stark.WebApplication) {
	ctx := context.Background()

	if application == nil || application.Application == nil {
		panic("webApplication is nil or application is nil")
	}
	// app instance once validate
	err := appInstanceOnceValidate()
	if err != nil {
		log.Errorf(ctx, "禁止重复创建应用:%v", err)
		return
	}

	application.Type = stark.AppTypeWeb
	stark.WebInstance = application

	err = runWeb(application)
	if err != nil {
		log.Errorf(ctx, "运行%s服务异常:%+v", stark.AppTypeMap[application.Type], err)
	}
}

// runWeb runs http and grpc application.
func runWeb(app *stark.WebApplication) error {
	var err error

	// 1. init application
	err = initApplication(app.Application)
	if err != nil {
		return err
	}

	// 2 init http and grpc vars
	err = setupWebVars(app)
	if err != nil {
		return err
	}

	// 配置http服务
	err = configHttpServer(app.ServerConfig)
	if err != nil {
		return err
	}

	// 注入http和grpc配置参数
	injectWebConfig(app)

	// 初始化框架适配器
	err = NewWebFrameworkAdapter(app.ServerConfig).Init()
	if err != nil {
		return err
	}

	return ioc.Run()
}

// 注入web配置参数
func injectWebConfig(app *stark.WebApplication) {
	ioc.Property("application.port", app.Port)
	ioc.Property("application.framework-strategy", int32(app.Strategy))
}

// setupWebVars ...
func setupWebVars(app *stark.WebApplication) error {
	serverOptions := &grpcModule.ServerOptions{}
	serverOptions.Options = append(serverOptions.Options, app.GrpcServerOptions...)

	if len(serverOptions.Options) > 0 {
		ioc.Object(serverOptions)
	}

	// 安装grpc服务
	setupGrpcServer()
	return nil
}
