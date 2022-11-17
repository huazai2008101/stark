package echo

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/huazai2008101/stark/base/log"
	"github.com/huazai2008101/stark/base/util"
	"github.com/huazai2008101/stark/ioc"
	"github.com/huazai2008101/stark/module/web"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ucarion/urlpath"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type EchoStarter struct {
	echo        *echo.Echo             `autowire:""`
	group       *echo.Group            `autowire:""`
	routerInits []web.RouteInitializer `autowire:"*?"`
	name        string                 `value:"${application.name}"`
	// 链路追踪服务器地址
	traceUrl    string                `value:"${trace.url:=}"`
	middlewares []echo.MiddlewareFunc `autowire:"*?"`
	// 不记录日志的路由
	excludeLogPaths []string `value:"${application.log.excludePath:=}"`
}

func NewEchoStarter() ioc.AppEvent {
	return &EchoStarter{}
}

func (s *EchoStarter) OnAppStart(ctx ioc.Context) {
	// 设置健康检查
	s.setHealthCheck()

	// 传递x-request-id参数
	s.group.Use(s.setRequestId)

	// 设置recover捕捉
	s.group.Use(s.recoverHandle)

	// 打印请求日志
	s.group.Use(s.logHandle())

	// 链路追踪设置
	s.setTraceProvider()

	// 允许跨域设置
	s.setAllowCors()

	// 在这里添加过滤器
	if len(s.middlewares) > 0 {
		s.group.Use(s.middlewares...)
	}

	// 最后一步初始化路由
	for _, v := range s.routerInits {
		v.Init()
	}
}

// 设置健康检查
func (s *EchoStarter) setHealthCheck() {
	s.echo.GET("/ping", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]interface{}{
			"message": "pong",
		})
	})
}

func (s *EchoStarter) setRequestId(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		ctx.SetRequest(ctx.Request().WithContext(web.BuildEchoContext(ctx)))
		return nil
	}
}

func (s *EchoStarter) recoverHandle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		defer func() {
			if panic := recover(); panic != nil {
				log.Errorf(ctx.Request().Context(), "panic:%v %s", panic, util.PanicStack())
				ctx.JSON(http.StatusBadRequest, map[string]interface{}{
					"message": fmt.Sprintf("服务器异常:%v", panic),
				})
			}
		}()

		return next(ctx)
	}
}

func (s *EchoStarter) logHandle() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		pathMatchers := make([]urlpath.Path, 0, len(s.excludeLogPaths))
		pathMatchers = append(pathMatchers, urlpath.New(fmt.Sprintf("/%s/ping", s.name)))
		pathMatchers = append(pathMatchers, urlpath.New(fmt.Sprintf("/%s/swagger/*", s.name)))
		for _, v := range s.excludeLogPaths {
			pathMatchers = append(pathMatchers, urlpath.New(fmt.Sprintf("/%s%s", s.name, v)))
		}

		return func(c echo.Context) error {
			for _, v := range pathMatchers {
				if _, ok := v.Match(c.Request().URL.Path); ok {
					// 匹配上则不需要打印日志
					next(c)
					return nil
				}
			}
			startTime := time.Now()
			var body []byte
			isJsonRequest := strings.Contains(strings.ToLower(c.Request().Header.Get("content-type")), "application/json")
			if c.Request().Method == http.MethodPost && isJsonRequest {
				body, _ = ioutil.ReadAll(c.Request().Body)
				c.Request().Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}
			next(c)
			endTime := time.Now()

			var buf bytes.Buffer
			buf.WriteString(fmt.Sprintf("method:%s status:%d duration:%s path:%s \n", c.Request().Method, c.Response().Status, endTime.Sub(startTime), c.Request().URL))

			if username := c.Get("username"); username != nil {
				buf.WriteString(fmt.Sprintf("用户信息: %v", username))
				if userNo := c.Get("userNo"); userNo != nil {
					buf.WriteString(fmt.Sprintf(" %v", userNo))
				}
				buf.WriteString("/n")
			}

			buf.WriteString("请求头: ")
			for k, v := range c.Request().Header {
				buf.WriteString(fmt.Sprintf("%s:%s ", k, strings.Join(v, ",")))
			}
			buf.WriteString("\n")

			if isJsonRequest && len(body) > 0 {
				buf.WriteString("body参数: ")
				buf.Write(body)
				buf.WriteString("/n")
			}

			if !isJsonRequest && c.Request().Method == http.MethodPost {
				buf.WriteString("post参数: ")
				for k, v := range c.Request().PostForm {
					buf.WriteString(fmt.Sprintf("%s:%s ", k, strings.Join(v, ",")))
				}
			}

			log.Info(web.BuildEchoContext(c), buf.String())
			return nil
		}

	}
}

func (s *EchoStarter) setTraceProvider() {
	if s.traceUrl == "" {
		return
	}
	ctx := context.Background()

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(s.traceUrl)))
	if err != nil {
		log.Errorf(ctx, "EchoStarter %s 初始化链路追踪异常:%+v", s.name, err)
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

	s.group.Use(otelecho.Middleware(s.name))
	log.Infof(ctx, "EchoStarter %s 已启用链路追踪功能", s.name)
}

// 允许跨域设置
func (s *EchoStarter) setAllowCors() {
	s.group.Use(middleware.CORS())
}

func (s *EchoStarter) OnAppStop(ctx context.Context) {
	s.echo.Shutdown(ctx)
}
