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
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	"github.com/huazai2008101/stark/base/log"
	"github.com/huazai2008101/stark/ioc/arg"
	"github.com/huazai2008101/stark/ioc/conf"
	"github.com/huazai2008101/stark/ioc/internal"
)

// AppRunner 命令行启动器接口
type AppRunner interface {
	Run(ctx Context)
}

// AppEvent 应用运行过程中的事件
type AppEvent interface {
	OnAppStart(ctx Context)        // 应用启动的事件
	OnAppStop(ctx context.Context) // 应用停止的事件
}

// App 应用
type App struct {
	c *container

	exitChan chan struct{}

	Events  []AppEvent  `autowire:"${application-event.collection:=*?}"`
	Runners []AppRunner `autowire:"${command-line-runner.collection:=*?}"`
}

// NewApp application 的构造函数
func NewApp() *App {
	return &App{
		c:        New().(*container),
		exitChan: make(chan struct{}),
	}
}

func (app *App) Run() error {

	// 响应控制台的 Ctrl+C 及 kill 命令。
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		sig := <-ch
		app.ShutDown(fmt.Sprintf("signal %v", sig))
	}()

	if err := app.start(); err != nil {
		return err
	}

	<-app.exitChan

	app.c.Close()
	log.Info(app.c.Context(), "application exited")
	return nil
}

func (app *App) clear() {
	app.c.clear()
}

func (app *App) start() error {

	app.Object(app)

	e := &configuration{
		p:               conf.New(),
		resourceLocator: new(defaultResourceLocator),
	}

	if err := e.prepare(); err != nil {
		return err
	}

	if err := app.loadProperties(e); err != nil {
		return err
	}

	// 保存从环境变量和命令行解析的属性
	for _, k := range e.p.Keys() {
		app.c.p.Set(k, e.p.Get(k))
	}

	if err := app.c.Refresh(internal.AutoClear(false)); err != nil {
		return err
	}

	// 执行命令行启动器
	for _, r := range app.Runners {
		r.Run(app.c)
	}

	// 通知应用启动事件
	for _, event := range app.Events {
		event.OnAppStart(app.c)
	}

	app.clear()

	// 通知应用停止事件
	app.c.Go(func(ctx context.Context) {
		<-ctx.Done()
		for _, event := range app.Events {
			event.OnAppStop(context.Background())
		}
	})

	log.Info(app.c.Context(), "application started successfully")
	return nil
}

func (app *App) loadProperties(e *configuration) error {
	var resources []Resource

	for _, ext := range e.ConfigExtensions {
		sources, err := app.loadResource(e, "application"+ext)
		if err != nil {
			return err
		}
		resources = append(resources, sources...)
	}

	for _, profile := range e.ActiveProfiles {
		for _, ext := range e.ConfigExtensions {
			sources, err := app.loadResource(e, "application-"+profile+ext)
			if err != nil {
				return err
			}
			resources = append(resources, sources...)
		}
	}

	for _, resource := range resources {
		b, err := ioutil.ReadAll(resource)
		if err != nil {
			return err
		}
		p, err := conf.Bytes(b, filepath.Ext(resource.Name()))
		if err != nil {
			return err
		}
		for _, key := range p.Keys() {
			app.c.p.Set(key, p.Get(key))
		}
	}

	return nil
}

func (app *App) loadResource(e *configuration, filename string) ([]Resource, error) {

	var locators []ResourceLocator
	locators = append(locators, e.resourceLocator)

	var resources []Resource
	for _, locator := range locators {
		sources, err := locator.Locate(filename)
		if err != nil {
			return nil, err
		}
		resources = append(resources, sources...)
	}
	return resources, nil
}

// ShutDown 关闭执行器
func (app *App) ShutDown(msg ...string) {
	log.Infof(app.c.Context(), "program will exit %s", strings.Join(msg, " "))
	select {
	case <-app.exitChan:
		// chan 已关闭，无需再次关闭。
	default:
		close(app.exitChan)
	}
}

// Go 参考 Container.Go 的解释。
func (app *App) Go(fn func(ctx context.Context)) {
	app.c.Go(fn)
}

// OnProperty 当 key 对应的属性值准备好后发送一个通知。
func (app *App) OnProperty(key string, fn interface{}) {
	app.c.OnProperty(key, fn)
}

// Property 参考 Container.Property 的解释。
func (app *App) Property(key string, value interface{}) {
	app.c.Property(key, value)
}

// Object 参考 Container.Object 的解释。
func (app *App) Object(i interface{}) *BeanDefinition {
	return app.c.register(NewBean(reflect.ValueOf(i)))
}

// Provide 参考 Container.Provide 的解释。
func (app *App) Provide(ctor interface{}, args ...arg.Arg) *BeanDefinition {
	return app.c.register(NewBean(ctor, args...))
}
