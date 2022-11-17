package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/huazai2008101/stark/base/cast"
	"github.com/huazai2008101/stark/base/log"
	"github.com/huazai2008101/stark/base/util"
	"github.com/huazai2008101/stark/discovery"
	"github.com/huazai2008101/stark/ioc"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type etcdServiceDiscovery struct {
	appName    string `value:"${application.name}"`
	appPort    int    `value:"${application.port}"`
	url        string `value:"${discovery.url:=127.0.0.1:2379}"`
	namespace  string `value:"${discovery.namespace:=default}"`
	localIP    string
	serviceMap sync.Map
	client     *clientv3.Client
}

func NewEtcdServiceDiscovery() discovery.ServiceDiscovery {
	return &etcdServiceDiscovery{}
}

// 初始化etcd客户端
func (s *etcdServiceDiscovery) OnInit(ctx ioc.Context) error {
	if s.client != nil {
		return nil
	}
	var err error
	s.client, err = clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(s.url, ","),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Errorf(ctx.Context(), "etcdServiceDiscovery 实例化etcd客户端异常:%+v", err)
		return err
	}

	// 先初始化现有服务信息，再监听增量服务信息
	err = s.initService()
	if err != nil {
		return err
	}
	go s.watchService()
	return nil
}

func (s *etcdServiceDiscovery) getLocalIP() string {
	if s.localIP != "" {
		return s.localIP
	}
	s.localIP = util.LocalIPv4()
	return s.localIP
}

func (s *etcdServiceDiscovery) Register() error {
	ctx := context.Background()

	key := fmt.Sprintf("%s/%s/%s:%d", s.getKeyPrefix(), s.appName, s.getLocalIP(), s.appPort)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	leaseClient := clientv3.NewLease(s.client)
	grantResp, err := leaseClient.Grant(ctx, 2)
	if err != nil {
		return errors.WithMessage(err, "grant fail")
	}

	val, _ := json.Marshal(discovery.ServiceInfo{
		Name:    s.appName,
		Address: s.getLocalIP(),
		Port:    s.appPort,
	})

	kvClient := clientv3.NewKV(s.client)
	_, err = kvClient.Put(ctx, key, string(val), clientv3.WithLease(grantResp.ID))
	if err != nil {
		return errors.WithMessage(err, "put fail")
	}

	ch, err := leaseClient.KeepAlive(context.Background(), grantResp.ID)
	if err != nil {
		return errors.WithMessage(err, "keepAlive fail")
	}

	go func() {
		for range ch {

		}
		log.Errorf(ctx, "etcdServiceDiscovery etcd连接已断开 %s", s.appName)
	}()

	log.Infof(ctx, "etcdServiceDiscovery %s服务注册成功(%s),etcd服务:%s", s.appName, fmt.Sprintf("%s:%d", s.getLocalIP(), s.appPort), s.url)
	return nil
}

// 监听服务动态
func (s *etcdServiceDiscovery) watchService() {
	watcherClient := clientv3.NewWatcher(s.client)
	ch := watcherClient.Watch(context.Background(), s.getKeyPrefix(), clientv3.WithPrefix())
	for v := range ch {
		s.updateService(v)
	}
}

// 初始化服务信息
func (s *etcdServiceDiscovery) initService() error {
	kvClient := clientv3.NewKV(s.client)
	kvResp, err := kvClient.Get(context.Background(), s.getKeyPrefix(), clientv3.WithPrefix())
	if err != nil {
		return errors.WithMessage(err, "init service info fail")
	}
	for _, v := range kvResp.Kvs {
		strArr := strings.Split(string(v.Key), "/")
		serviceName := strArr[2]
		instanceSet, ok := s.serviceMap.Load(serviceName)
		if !ok {
			instanceSet = discovery.NewServiceSet()
		}
		var serviceInfo discovery.ServiceInfo
		err = json.Unmarshal(v.Value, &serviceInfo)
		if err != nil {
			log.Errorf(context.Background(), "etcdServiceDiscovery 反序列化服务信息异常:%+v %s", err, v.Value)
			continue
		}
		s.serviceMap.Store(serviceName, instanceSet)
	}
	return nil
}

func (s *etcdServiceDiscovery) getKeyPrefix() string {
	return fmt.Sprintf("discovery/%s", s.namespace)
}

// 更新注册服务信息
func (s *etcdServiceDiscovery) updateService(watchResp clientv3.WatchResponse) {
	ctx := context.Background()

	var err error
	for _, e := range watchResp.Events {
		strArr := strings.Split(string(e.Kv.Key), "/")
		serviceName := strArr[2]
		var instanceSet *discovery.ServiceSet
		val, ok := s.serviceMap.Load(serviceName)
		if !ok {
			instanceSet = discovery.NewServiceSet()
		} else {
			instanceSet = val.(*discovery.ServiceSet)
		}

		var serviceInfo discovery.ServiceInfo
		if e.Type == mvccpb.PUT {
			err = json.Unmarshal(e.Kv.Value, &serviceInfo)
			if err != nil {
				log.Errorf(ctx, "etcdServiceDiscovery 反序列化服务信息异常:%+v %s", err, e.Kv.Value)
				continue
			}
			if instanceSet.Put(serviceInfo) {
				log.Infof(ctx, "etcdServiceDiscovery 新增服务:%s %s:%d", serviceInfo.Name, serviceInfo.Address, serviceInfo.Port)
			}
		}

		if e.Type == mvccpb.DELETE {
			endpoint := strings.Split(strArr[3], ":")
			serviceInfo = discovery.ServiceInfo{
				Name:    serviceName,
				Address: endpoint[0],
				Port:    cast.ToInt(endpoint[1]),
			}
			log.Infof(ctx, "etcdServiceDiscovery 移除服务:%s %s:%d", serviceInfo.Name, serviceInfo.Address, serviceInfo.Port)

			// 如果是因为etcd服务挂掉会导致续约终止从而导致自身服务注册信息被移除，需要进行重新注册服务
			if serviceInfo.Address == s.getLocalIP() && serviceInfo.Port == s.appPort {
				go func() {
					// 等待一秒钟后再进行重新注册，避免etcd注册服务事件比删除服务事件先到达客户端
					time.Sleep(time.Second)
					s.Register()
				}()
			}
		}
		s.serviceMap.Store(serviceName, instanceSet)
	}
}

// 已注册服务实例
func (s *etcdServiceDiscovery) ServiceInstances(serviceName string) []discovery.ServiceInfo {
	val, ok := s.serviceMap.Load(serviceName)
	if ok {
		serviceInstance := val.(*discovery.ServiceSet)
		return serviceInstance.List()
	}

	log.Warnf(context.Background(), "etcdServiceDiscovery %s 服务实例不存在", s.appName)
	return nil
}

func (s *etcdServiceDiscovery) SchemeName() string {
	return "etcd"
}

func (s *etcdServiceDiscovery) SchemeUrl() string {
	return fmt.Sprintf("%s://%s", s.SchemeName(), s.url)
}
