package gin

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/huazai2008101/stark/base/log"
	"github.com/huazai2008101/stark/base/util"
	"github.com/huazai2008101/stark/ioc"
	"github.com/huazai2008101/stark/module/web"
	"github.com/ucarion/urlpath"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type GinStarter struct {
	gin         *gin.Engine            `autowire:""`
	group       *gin.RouterGroup       `autowire:""`
	routerInits []web.RouteInitializer `autowire:"*?"`
	name        string                 `value:"${application.name}"`
	// 链路追踪服务器地址
	traceUrl string `value:"${trace.url:=}"`
	// 用户自定义中间件
	middlewares []gin.HandlerFunc `autowire:"*?"`
	// 不记录日志的路由
	excludeLogPaths []string `value:"${application.log.excludePath:=}"`
}

func NewGinStarter() ioc.AppEvent {
	return &GinStarter{}
}

func (s *GinStarter) OnAppStart(ctx ioc.Context) {
	gin.SetMode(gin.ReleaseMode)

	// 传递x-request-id参数
	s.group.Use(s.setRequestId)

	// 设置健康检查
	s.setHealthCheck()

	// 设置recover捕捉
	s.group.Use(s.recoverHandle)

	// 链路追踪设置
	s.setTraceProvider()

	// 打印请求日志
	s.group.Use(s.logHandle())

	// 允许跨域设置
	s.setAllowCors()

	// 用户自定义中间件
	if len(s.middlewares) > 0 {
		s.group.Use(s.middlewares...)
	}

	// 最后一步初始化路由
	for _, v := range s.routerInits {
		v.Init()
	}
}

func (s *GinStarter) setRequestId(c *gin.Context) {
	c.Request = c.Request.WithContext(web.BuildGinContext(c))
}

func (s *GinStarter) logHandle() gin.HandlerFunc {
	pathMatchers := make([]urlpath.Path, 0, len(s.excludeLogPaths))
	pathMatchers = append(pathMatchers, urlpath.New(fmt.Sprintf("/%s/ping", s.name)))
	pathMatchers = append(pathMatchers, urlpath.New(fmt.Sprintf("/%s/swagger/*", s.name)))
	for _, v := range s.excludeLogPaths {
		pathMatchers = append(pathMatchers, urlpath.New(fmt.Sprintf("/%s%s", s.name, v)))
	}
	return func(c *gin.Context) {
		for _, v := range pathMatchers {
			if _, ok := v.Match(c.Request.URL.Path); ok {
				// 匹配上则不需要打印日志
				return
			}
		}
		startTime := time.Now()
		var body []byte
		isJsonRequest := strings.Contains(strings.ToLower(c.GetHeader("content-type")), "application/json")
		if c.Request.Method == http.MethodPost && isJsonRequest {
			body, _ = ioutil.ReadAll(c.Request.Body)
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		}
		c.Next()
		endTime := time.Now()

		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("method:%s status:%d duration:%s path:%s \n", c.Request.Method, c.Writer.Status(), endTime.Sub(startTime), c.Request.URL))

		if username, ok := c.Get("username"); ok {
			buf.WriteString(fmt.Sprintf("用户信息: %v", username))
			if userNo, ok := c.Get("userNo"); ok {
				buf.WriteString(fmt.Sprintf(" %v", userNo))
			}
			buf.WriteString("/n")
		}

		buf.WriteString("请求头: ")
		for k, v := range c.Request.Header {
			buf.WriteString(fmt.Sprintf("%s:%s ", k, strings.Join(v, ",")))
		}
		buf.WriteString("\n")

		if isJsonRequest && len(body) > 0 {
			buf.WriteString("body参数: ")
			buf.Write(body)
			buf.WriteString("/n")
		}

		if !isJsonRequest && c.Request.Method == http.MethodPost {
			buf.WriteString("post参数: ")
			for k, v := range c.Request.PostForm {
				buf.WriteString(fmt.Sprintf("%s:%s ", k, strings.Join(v, ",")))
			}
		}

		log.Info(web.BuildGinContext(c), buf.String())
	}
}

func (s *GinStarter) recoverHandle(c *gin.Context) {
	defer func() {
		if panic := recover(); panic != nil {
			log.Errorf(c.Request.Context(), "panic:%v %s", panic, util.PanicStack())
			c.JSON(http.StatusBadRequest, gin.H{
				"message": fmt.Sprintf("服务器异常:%v", panic),
			})
			c.Abort()
		}
	}()
	c.Next()
}

func (s *GinStarter) OnAppStop(ctx context.Context) {

}

// 设置健康检查
func (s *GinStarter) setHealthCheck() {
	s.gin.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
}

func (s *GinStarter) setTraceProvider() {
	if s.traceUrl == "" {
		return
	}
	ctx := context.Background()

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(s.traceUrl)))
	if err != nil {
		log.Errorf(ctx, "GinStarter %s 初始化链路追踪异常:%+v", s.name, err)
		return
	}

	batchSpanProcessor := trace.NewBatchSpanProcessor(exp)
	tracerProvider := trace.NewTracerProvider(
		trace.WithSpanProcessor(batchSpanProcessor),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("test"),
		)),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	s.group.Use(otelgin.Middleware(s.name))
	log.Infof(ctx, "GinStarter %s 已启用链路追踪功能", s.name)
}

// 允许跨域设置
func (s *GinStarter) setAllowCors() {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	s.group.Use(cors.New(config))
}
