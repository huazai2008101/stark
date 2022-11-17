/*
 * Copyright 2012-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ioc

import (
	"context"
	"os"
	"reflect"
	"strings"

	"github.com/huazai2008101/stark/base/log"
	"github.com/huazai2008101/stark/base/util"
	"github.com/huazai2008101/stark/ioc/arg"
)

func init() {
	// 如果发现是调试模式则设置日志级别为 Debug 级别。
	{
		s := os.Getenv("CGO_CFLAGS")
		if strings.Contains(s, "-O0") && strings.Contains(s, "-g") {
			log.SetLevel(log.DebugLevel)
		}
	}
}

var (
	gApp *App
)

func app() *App {
	if gApp == nil {
		gApp = NewApp()
	}
	return gApp
}

// Setenv 封装 os.Setenv 函数，如果发生 error 会 panic 。
func SetEnv(key string, value string) {
	err := os.Setenv(key, value)
	util.Panic(err).When(err != nil)
}

// Go 参考 App.Go 的解释。
func Go(fn func(ctx context.Context)) {
	app().Go(fn)
}

// Run 启动程序。
func Run() error {
	return app().Run()
}

// ShutDown 停止程序。
func ShutDown(msg ...string) {
	app().ShutDown(msg...)
}

// OnProperty 参考 App.OnProperty 的解释。
func OnProperty(key string, fn interface{}) {
	app().OnProperty(key, fn)
}

// Property 参考 Container.Property 的解释。
func Property(key string, value interface{}) {
	app().Property(key, value)
}

// Object 参考 Container.Object 的解释。
func Object(i interface{}) *BeanDefinition {
	return app().c.register(NewBean(reflect.ValueOf(i)))
}

// Provide 参考 Container.Provide 的解释。
func Provide(ctor interface{}, args ...arg.Arg) *BeanDefinition {
	return app().c.register(NewBean(ctor, args...))
}
