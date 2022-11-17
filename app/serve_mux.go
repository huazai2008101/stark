package app

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huazai2008101/stark"
	grpcModule "github.com/huazai2008101/stark/module/grpc"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// 多路复用器
type ServeMux struct {
	// 应用名称
	name string `value:"${application.name}"`
	// 应用类型
	appType int32                  `value:"${application.type}"`
	echo    *echo.Echo             `autowire:"?"`
	gin     *gin.Engine            `autowire:"?"`
	grpc    *grpcModule.GrpcServer `autowire:"?"`
	handler http.Handler
}

func (s *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *ServeMux) Init() {
	switch stark.AppType(s.appType) {
	case stark.AppTypeWeb:
		// http和grpc混合应用
		s.handler = h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
				s.grpc.Server.ServeHTTP(w, r)
			} else {
				s.httpHandle(w, r)
			}
		}), &http2.Server{})
	case stark.AppTypeHttp:
		// 纯http应用
		s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.httpHandle(w, r)
		})
	case stark.AppTypeGrpc:
		// 纯grpc应用
		s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.grpc.Server.ServeHTTP(w, r)
		})
	default:
		s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("unknow appType!"))
		})
	}

	originHandler := s.handler
	s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.RequestURI, s.rootPath()+"/debug/pprof/") {
			s.pprofHandle(w, r)
			return
		}
		originHandler.ServeHTTP(w, r)
	})
}

func (s *ServeMux) rootPath() string {
	return "/" + s.name
}

// Pprof
func (s *ServeMux) pprofHandle(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = strings.TrimPrefix(r.URL.Path, s.rootPath())
	r.RequestURI = strings.TrimPrefix(r.RequestURI, s.rootPath())
	uri := r.URL.Path
	if uri == "/debug/pprof/cmdline" {
		pprof.Cmdline(w, r)
	} else if uri == "/debug/pprof/profile" {
		pprof.Profile(w, r)
	} else if uri == "/debug/pprof/symbol" {
		pprof.Symbol(w, r)
	} else if uri == "/debug/pprof/trace" {
		pprof.Trace(w, r)
	} else if strings.HasPrefix(uri, "/debug/pprof/") {
		pprof.Index(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 not found!"))
	}
}

func (s *ServeMux) httpHandle(w http.ResponseWriter, r *http.Request) {
	// 判断是否启用gin框架，如果启用则把请求转发给gin
	if s.gin != nil {
		s.gin.ServeHTTP(w, r)
		return
	}
	// 判断是否启用echo框架，如果启用则把请求转发给echo
	if s.echo != nil {
		s.echo.ServeHTTP(w, r)
		return
	}
	// 找不到路由则返回404
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 not found!"))
}

func NewServeMux() *ServeMux {
	instance := &ServeMux{}
	return instance
}
