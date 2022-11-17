package web

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/huazai2008101/stark"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/metadata"
)

// 构建gin上下文对象
func BuildGinContext(ginCtx *gin.Context) context.Context {
	ctx := ginCtx.Request.Context()

	md := make(metadata.MD)
	requestId := ginCtx.Request.Header.Get(stark.MetadataRequestId)
	if requestId != "" {
		md.Set(stark.MetadataRequestId, requestId)
	}
	if md.Len() > 0 {
		ctx = metadata.NewIncomingContext(ctx, md)
	}
	return ctx
}

// 构建echo上下文对象
func BuildEchoContext(echoCtx echo.Context) context.Context {
	ctx := echoCtx.Request().Context()

	md := make(metadata.MD)
	requestId := echoCtx.Request().Header.Get(stark.MetadataRequestId)
	if requestId != "" {
		md.Set(stark.MetadataRequestId, requestId)
	}
	if md.Len() > 0 {
		ctx = metadata.NewIncomingContext(ctx, md)
	}
	return ctx
}
