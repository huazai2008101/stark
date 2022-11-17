package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/huazai2008101/stark"
	"github.com/huazai2008101/stark/base/cast"
	"github.com/huazai2008101/stark/base/log"
	"github.com/huazai2008101/stark/base/util"
	"github.com/huazai2008101/stark/discovery"
	"github.com/huazai2008101/stark/ioc"
)

type consulServiceDiscovery struct {
	appName    string `value:"${application.name}"`
	appPort    int    `value:"${application.port}"`
	appType    int32  `value:"${application.type}"`
	url        string `value:"${discovery.url:=127.0.0.1:8500}"`
	namespace  string `value:"${discovery.namespace:=}"`
	serviceMap sync.Map
	client     *api.Client
	watchers   map[string]*watch.Plan
}

func NewConsulServiceDiscovery() discovery.ServiceDiscovery {
	return &consulServiceDiscovery{
		watchers: make(map[string]*watch.Plan),
	}
}

// 初始化consul客户端
func (s *consulServiceDiscovery) OnInit(ctx ioc.Context) error {
	if s.client != nil {
		return nil
	}
	config := api.DefaultConfig()
	config.Address = s.url
	var err error
	s.client, err = api.NewClient(config)
	if err != nil {
		log.Errorf(ctx.Context(), "consulServiceDiscovery 实例化consul客户端异常:%+v", err)
		return err
	}
	return nil
}

func (s *consulServiceDiscovery) Register() error {
	ctx := context.Background()

	agent := s.client.Agent()
	ipv4 := util.LocalIPv4()

	endpoit := fmt.Sprintf("%s:%d", ipv4, s.appPort)
	reg := &api.AgentServiceRegistration{
		ID:        endpoit,
		Name:      s.appName,
		Port:      s.appPort,
		Address:   ipv4,
		Namespace: s.namespace,
		Tags: []string{
			s.appName,
			ipv4,
			stark.AppTypeMap[stark.AppType(s.appType)],
		},
		Meta: map[string]string{
			"appType": cast.ToString(s.appType),
		},
		Check: &api.AgentServiceCheck{
			Interval:                       "3s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "300s",
			HTTP:                           fmt.Sprintf("http://%s/ping", endpoit),
		},
	}
	err := agent.ServiceRegister(reg)
	if err != nil {
		log.Errorf(ctx, "consulServiceDiscovery %s 注册服务异常:%+v endpoint:%s", s.appName, err, endpoit)
		return err
	}
	log.Infof(ctx, "consulServiceDiscovery %s服务注册成功(%s),consul服务:%s", s.appName, endpoit, s.url)

	return nil
}

// 监听服务动态
func (s *consulServiceDiscovery) watchService(name string) {
	ctx := context.Background()

	w, ok := s.watchers[name]
	// 存在并且没有关闭状态则直接return
	if ok && !w.IsStopped() {
		return
	}
	// watch endpoint 的请求参数，具体见官方文档：https://www.consul.io/docs/dynamic-app-config/watches#service
	wp, err := watch.Parse(map[string]interface{}{
		"type":    "service",
		"service": name,
	})
	if err != nil {
		log.Errorf(ctx, "consulServiceDiscovery %s 初始化监听服务异常:%+v", name, err)
		return
	}

	// 定义service变化后所执行的程序(函数)handler
	wp.Handler = func(idx uint64, data interface{}) {
		switch d := data.(type) {
		case []*api.ServiceEntry:
			for _, i := range d {
				val, ok := s.serviceMap.Load(i.Service.Service)
				if !ok {
					s.initService(i.Service.Service)
					continue
				}
				serviceSet := val.(*discovery.ServiceSet)
				switch i.Checks.AggregatedStatus() {
				case api.HealthPassing:
					ok = serviceSet.Put(discovery.ServiceInfo{
						Name:    i.Service.Service,
						Address: i.Service.Address,
						Port:    i.Service.Port,
					})
					if ok {
						log.Infof(ctx, "consulServiceDiscovery 新增服务:%s %s:%d node:%s", i.Service.Service, i.Service.Address, i.Service.Port, i.Node.Address)
					}

				default:
					ok = serviceSet.Remove(discovery.ServiceInfo{
						Name:    i.Service.Service,
						Address: i.Service.Address,
						Port:    i.Service.Port,
					})
					if ok {
						log.Infof(ctx, "consulServiceDiscovery 移除服务(%s):%s %s:%d node:%s", i.Checks.AggregatedStatus(), i.Service.Service, i.Service.Address, i.Service.Port, i.Node.Address)
					}
				}
			}
		}
	}
	// 启动监控
	go wp.Run(s.url)
	// 对已启动监控的service作一个记录
	s.watchers[name] = wp
	log.Infof(ctx, "consulServiceDiscovery %s 动态监听服务启动完毕", name)
}

// 已注册服务实例
func (s *consulServiceDiscovery) ServiceInstances(serviceName string) []discovery.ServiceInfo {
	val, ok := s.serviceMap.Load(serviceName)
	if ok {
		serviceInstance := val.(*discovery.ServiceSet)
		return serviceInstance.List()
	}

	// 如果服务不存在重新从consul初始化
	s.initService(serviceName)
	val, ok = s.serviceMap.Load(serviceName)
	if ok {
		serviceInstance := val.(*discovery.ServiceSet)
		return serviceInstance.List()
	}
	return nil
}

func (s *consulServiceDiscovery) initService(name string) {
	ctx := context.Background()

	_, agentServices, err := s.client.Agent().AgentHealthServiceByNameOpts(name, &api.QueryOptions{
		Namespace: s.namespace,
	})
	if err != nil {
		log.Errorf(ctx, "consulServiceDiscovery %s 初始化服务异常:%+v", s.appName, err)
		return
	}
	if len(agentServices) == 0 {
		return
	}

	var instanceSet *discovery.ServiceSet
	val, ok := s.serviceMap.Load(name)
	if !ok {
		instanceSet = discovery.NewServiceSet()
	} else {
		instanceSet = val.(*discovery.ServiceSet)
	}
	for _, v := range agentServices {
		if v.AggregatedStatus != api.HealthPassing {
			continue
		}
		instanceSet.Put(discovery.ServiceInfo{
			Name:    v.Service.Service,
			Address: v.Service.Address,
			Port:    v.Service.Port,
		})
	}
	if len(instanceSet.List()) == 0 {
		for _, v := range agentServices {
			instanceSet.Put(discovery.ServiceInfo{
				Name:    v.Service.Service,
				Address: v.Service.Address,
				Port:    v.Service.Port,
			})
		}
	}
	temp, _ := json.Marshal(instanceSet.List())
	log.Infof(ctx, "consulServiceDiscovery %s 初始化服务实例:%s", name, temp)
	s.serviceMap.Store(name, instanceSet)
	s.watchService(name)
}

func (s *consulServiceDiscovery) SchemeName() string {
	return "consul"
}

func (s *consulServiceDiscovery) SchemeUrl() string {
	return fmt.Sprintf("%s://%s", s.SchemeName(), s.url)
}
