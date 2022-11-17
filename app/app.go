package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/extra/redisotel"
	"github.com/go-redis/redis/v8"
	"github.com/huazai2008101/stark"
	"github.com/huazai2008101/stark/base/log"
	"github.com/huazai2008101/stark/ioc"
	"github.com/jojo-jie/otelgorm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	flagEnv = flag.String("env", "", "set exec environment eg: dev,test,prod")
)

var (
	appInstanceOnce    int32
	errAppInstanceOnce = errors.New("the same app type can only be registered once")
)

func appInstanceOnceValidate() error {
	ok := atomic.CompareAndSwapInt32(&appInstanceOnce, 0, 1)
	if !ok {
		return errAppInstanceOnce
	}
	return nil
}

func initApplication(application *stark.Application) error {
	// 验证应用数据
	err := validateApplication(application)
	if err != nil {
		return err
	}

	// 显示应用版本
	showAppVersion(application)

	// 初始化运行环境
	initRuntimeEnv(application)

	// 注入应用配置信息
	injectApplicationConfig(application)

	// 服务发现适配器初始化
	if application.Discovery != nil {
		err = NewDiscoveryAdapter(application.Discovery).Init()
		if err != nil {
			return err
		}
	}

	// 安装组件
	err = setupCommonVars(application)
	if err != nil {
		return err
	}

	// 安装用户自定义组件
	if application.SetupVars != nil {
		err = application.SetupVars()
		if err != nil {
			return fmt.Errorf("application.SetupVars err: %v", err)
		}
	}
	return nil
}

func validateApplication(application *stark.Application) error {
	if application.Name == "" {
		return fmt.Errorf("应用名称不能为空")
	}
	return nil
}

// 初始化运行环境
func initRuntimeEnv(application *stark.Application) {
	if application.Environment != "" {
		return
	}
	// 运行环境
	if *flagEnv != "" {
		application.Environment = *flagEnv
		return
	}
	application.Environment = os.Getenv("env")
}

// setupCommonVars setup application global vars.
func setupCommonVars(application *stark.Application) error {
	// 安装数据库组件
	err := setupDatabase(application)
	if err != nil {
		return err
	}

	return nil
}

// 安装各种数据库组件
func setupDatabase(application *stark.Application) error {
	if len(application.DbConns) == 0 {
		return nil
	}

	var err error
	for _, v := range application.DbConns {
		switch v.Type {
		case stark.DbTypeMyql:
			err = setupMysql(application, v)
		case stark.DbTypeRedis:
			err = setupRedis(v)
		}
		if err != nil {
			return fmt.Errorf("安装数据库组件异常:%+v", err)
		}
	}

	return nil
}

// 安装mysql
func setupMysql(application *stark.Application, info stark.DbConnInfo) error {
	ctx := context.Background()

	gormConf := &gorm.Config{}
	if application.IsDebug {
		gormConf.Logger = logger.Default.LogMode(logger.Info)
	}
	db, err := gorm.Open(mysql.Open(info.Url), gormConf)
	if err != nil {
		log.Errorf(ctx, "连接%s数据库异常:%+v", stark.DbTypeText[info.Type], err)
		return err
	}
	// 设置数据库连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Errorf(ctx, "设置%s数据库连接池异常:%+v", stark.DbTypeText[info.Type], err)
		return err
	}

	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	var size int
	maxIdleConn := 10
	if val, ok := info.Extras["maxIdleConn"]; ok {
		size = val.(int)
		if size > 0 {
			maxIdleConn = size
		}
	}
	sqlDB.SetMaxIdleConns(maxIdleConn)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	maxOpenConn := 1000
	if val, ok := info.Extras["maxOpenConn"]; ok {
		size = val.(int)
		if size > 0 {
			maxOpenConn = size
		}
	}
	sqlDB.SetMaxOpenConns(maxOpenConn)

	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	connMaxLifetime := time.Hour
	if val, ok := info.Extras["connMaxLifetime"]; ok {
		connMaxLifetime = val.(time.Duration)
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	if stark.IsEnableTrace {
		plugin := otelgorm.NewPlugin(otelgorm.WithServiceName("gorm"))
		err = db.Use(plugin)
		if err != nil {
			log.Errorf(ctx, "%s数据库设置链路追踪异常:%+v", stark.DbTypeText[info.Type], err)
			return err
		}
	}

	ioc.Object(db).Name(info.Name).Destroy(func(db *gorm.DB) {
		err = sqlDB.Close()
		if err != nil {
			log.Errorf(ctx, "关闭%s数据库连接异常:%+v", stark.DbTypeText[info.Type], err)
		}
	})
	return nil
}

// 安装redis
func setupRedis(info stark.DbConnInfo) error {
	var username, password string
	var db int
	var dialTimeout, readTimeout, writeTimeout, idleTimeout int
	if val, ok := info.Extras["username"]; ok {
		username = val.(string)
	}
	if val, ok := info.Extras["password"]; ok {
		password = val.(string)
	}
	if val, ok := info.Extras["db"]; ok {
		db = val.(int)
	}
	if val, ok := info.Extras["dialTimeout"]; ok {
		dialTimeout = val.(int)
	}
	if val, ok := info.Extras["readTimeout"]; ok {
		readTimeout = val.(int)
	}
	if val, ok := info.Extras["writeTimeout"]; ok {
		writeTimeout = val.(int)
	}
	if val, ok := info.Extras["idleTimeout"]; ok {
		idleTimeout = val.(int)
	}
	client := redis.NewClient(&redis.Options{
		Addr:         info.Url,
		Username:     username,
		Password:     password,
		DB:           db,
		DialTimeout:  time.Duration(dialTimeout) * time.Second,
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
		IdleTimeout:  time.Duration(idleTimeout) * time.Second,
	})
	if stark.IsEnableTrace {
		client.AddHook(redisotel.TracingHook{})
	}
	ioc.Object(client)
	return nil
}

func showAppVersion(app *stark.Application) {
	var logo = `%20ad88888ba%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%2088%20%20%20%20%20%20%20%20%20%0Ad8%22%20%20%20%20%20%228b%20%20%2Cd%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%2088%20%20%20%20%20%20%20%20%20%0AY8%2C%20%20%20%20%20%20%20%20%20%2088%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%2088%20%20%20%20%20%20%20%20%20%0A%60Y8aaaaa%2C%20%20MM88MMM%20%20%2CadPPYYba%2C%20%208b%2CdPPYba%2C%20%2088%20%20%20%2Cd8%20%20%20%0A%20%20%60%22%22%22%22%228b%2C%20%2088%20%20%20%20%20%22%22%20%20%20%20%20%60Y8%20%2088P'%20%20%20%22Y8%20%2088%20%2Ca8%22%20%20%20%20%0A%20%20%20%20%20%20%20%20%608b%20%2088%20%20%20%20%20%2CadPPPPP88%20%2088%20%20%20%20%20%20%20%20%20%208888%5B%20%20%20%20%20%20%0AY8a%20%20%20%20%20a8P%20%2088%2C%20%20%20%2088%2C%20%20%20%20%2C88%20%2088%20%20%20%20%20%20%20%20%20%2088%60%22Yba%2C%20%20%20%0A%20%22Y88888P%22%20%20%20%22Y888%20%20%60%228bbdP%22Y8%20%2088%20%20%20%20%20%20%20%20%20%2088%20%20%20%60Y8a%20%20`
	var version = `[Major Version：%v Type：%v]`
	logoS, _ := url.QueryUnescape(logo)
	fmt.Println(logoS)
	fmt.Println("")
	fmt.Println(fmt.Sprintf(version, stark.Version, stark.AppTypeMap[app.Type]))
}

// 配置http服务
func configHttpServer(config *stark.ServerConfig) error {
	ctx := context.Background()

	// 注入多路复用器
	ioc.Provide(NewServeMux)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		log.Errorf(ctx, "监听服务端口异常:%+v port:%d", err, config.Port)
		return err
	}
	if config.Port == 0 {
		config.Port = l.Addr().(*net.TCPAddr).Port
	}
	ioc.Object(l)

	server := &http.Server{
		Handler:      nil,
		ReadTimeout:  time.Duration(config.ReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(config.WriteTimeout) * time.Millisecond,
	}
	ioc.Object(server)

	// 注入http启动器
	ioc.Provide(NewHttpStarter).Name("httpStarter").Order(10000)
	return nil
}

// 注入应用配置参数
func injectApplicationConfig(app *stark.Application) {
	ioc.Property("application.name", app.Name)
	ioc.Property("application.type", int32(app.Type))

	// 注入链路追踪配置
	if app.TraceUrl != "" {
		ioc.Property("trace.url", app.TraceUrl)
		stark.IsEnableTrace = true
	}
}
