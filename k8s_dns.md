# k8s dns

## 1. dns架构演进

k8s 1.4前的dns，有4个container组成， kube2sky, skyDNS, healthz, etcd。

* kube2sky通过k8s API监视k8s service资源的变化，并根据service的信息生成 DNS 记录写入到etcd中。

* skydns为集群中的 pod 提供 DNS 查询服务，DNS记录从etcd中读取。

* exec healthz提供健康检查功能。

![图](https://www.kubernetes.org.cn/img/2016/10/20161028145508.jpg)

在 k8s 1.4版本后，dns进行重新梳理

![图](https://www.kubernetes.org.cn/img/2016/10/20161028145516.jpg)

### kubedns
* 监视k8s Service资源并更新DNS记录
* 替换etcd，使用TreeCache数据结构保存DNS记录并实现SkyDNS的Backend接口
* 接入SkyDNS，对dnsmasq提供DNS查询服务

### dnsmasq
* 对集群提供DNS查询服务
* 设置kubedns为upstream
* 提供DNS缓存，降低kubedns负载，提高性能

dnsmasq能够缓存外部DNS记录，同时提供本地DNS解析或者作为外部DNS的代理，即dnsmasq会首先查找/etc/hosts等本地解析文件，然后再查找/etc/resolv.conf等外部nameserver配置文件中定义的外部DNS。所以说dnsmasq是一个很不错的DNS中继。DNS配置同样写入dnsmasq.conf配置文件里。


### exechealthz
* 定期检查kubedns和dnsmasq的健康状态
* 为k8s活性检测提供HTTP API


## 2. 代码结构图

![kubedns](http://www.do1618.com/wp-content/uploads/2017/05/kubeDNS.png)


### k8s client-go 流程分析

KubeDNS 启动参数：

```
/kube-dns--domain=cluster.local. --dns-port=10053 --config-map=kube-dns --v=0
```

创建 KubeDNS 对象，完成相关初始化工作

```go
func NewKubeDNS(client clientset.Interface, clusterDomain string, timeout time.Duration, configSync config.Sync) *KubeDNS {
	kd := &KubeDNS{
		kubeClient:          client,
		domain:              clusterDomain,
		cache:               treecache.NewTreeCache(),  // 用于保存 dns 记录的 TreeCache 结构
		cacheLock:           sync.RWMutex{},
		nodesStore:          kcache.NewStore(kcache.MetaNamespaceKeyFunc), // 获取 k8s nodes 的接口
		reverseRecordMap:    make(map[string]*skymsg.Service),
		clusterIPServiceMap: make(map[string]*v1.Service),
		domainPath:          util.ReverseArray(strings.Split(strings.TrimRight(clusterDomain, "."), ".")),
		initialSyncTimeout:  timeout,

		configLock: sync.RWMutex{},
		configSync: configSync,
	}

	kd.setEndpointsStore()  // 获取 endpoints 信息的对象
	kd.setServicesStore()   // 获取 service 信息的对象

	return kd
}

``` 

获取 EndpointsStore 对象


```
func (kd *KubeDNS) setEndpointsStore() {
	// Returns a cache.ListWatch that gets all changes to endpoints.
	// 从 kcache.NewInformer 中返回 epStore 和 epsController
	// kcache "k8s.io/client-go/tools/cache"

	kd.endpointsStore, kd.endpointsController = kcache.NewInformer(
		&kcache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return kd.kubeClient.Core().Endpoints(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return kd.kubeClient.Core().Endpoints(v1.NamespaceAll).Watch(options)
			},
		},
		&v1.Endpoints{},
		resyncPeriod,   // resyncPeriod = 5 * time.Minute
		kcache.ResourceEventHandlerFuncs{
			AddFunc:    kd.handleEndpointAdd,    // Add endpoint 的处理方式
			UpdateFunc: kd.handleEndpointUpdate, // Update endpoint 的处理方式
			// If Service is named headless need to remove the reverse dns entries.
			DeleteFunc: kd.handleEndpointDelete, // delete enpoint 的处理方式
		},
	)
}

// 获取 ServicesStore 对象

func (kd *KubeDNS) setServicesStore() {
	// Returns a cache.ListWatch that gets all changes to services.
	kd.servicesStore, kd.serviceController = kcache.NewInformer(
		&kcache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return kd.kubeClient.Core().Services(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return kd.kubeClient.Core().Services(v1.NamespaceAll).Watch(options)
			},
		},
		&v1.Service{},
		resyncPeriod,
		kcache.ResourceEventHandlerFuncs{
			AddFunc:    kd.newService,
			DeleteFunc: kd.removeService,
			UpdateFunc: kd.updateService,
		},
	)
}

```

NewInformer client-go 获取 相关对象

```
// NewInformer returns a Store and a controller for populating the store
// while also providing event notifications. You should only used the returned
// Store for Get/List operations; Add/Modify/Deletes will cause the event
// notifications to be faulty.
//
// Parameters:
//  * lw is list and watch functions for the source of the resource you want to
//    be informed of.
//  * objType is an object of the type that you expect to receive.
//  * resyncPeriod: if non-zero, will re-list this often (you will get OnUpdate
//    calls, even if nothing changed). Otherwise, re-list will be delayed as
//    long as possible (until the upstream source closes the watch or times out,
//    or you stop the controller).
//  * h is the object you want notifications sent to.
//
func NewInformer(
	lw ListerWatcher,
	objType runtime.Object,
	resyncPeriod time.Duration,  
	h ResourceEventHandler,
) (Store, *Controller) {
```

```go
func (kd *KubeDNS) newService(obj interface{}) {
	if service, ok := assertIsService(obj); ok {
		glog.V(2).Infof("New service: %v", service.Name)
		glog.V(4).Infof("Service details: %v", service)

		// ExternalName services are a special kind that return CNAME records
		if service.Spec.Type == v1.ServiceTypeExternalName {
			kd.newExternalNameService(service)
			return
		}
		// if ClusterIP is not set, a DNS entry should not be created
		if !v1.IsServiceIPSet(service) {
			kd.newHeadlessService(service)
			return
		}
		if len(service.Spec.Ports) == 0 {
			glog.Warningf("Service with no ports, this should not have happened: %v",
				service)
		}
		kd.newPortalService(service) // 最终会调用 kd.newPortalService
	}
}

```

```
func (kd *KubeDNS) newPortalService(service *v1.Service) {
	subCache := treecache.NewTreeCache()
	recordValue, recordLabel := util.GetSkyMsg(service.Spec.ClusterIP, 0)
	subCache.SetEntry(recordLabel, recordValue, kd.fqdn(service, recordLabel))

	// Generate SRV Records
	for i := range service.Spec.Ports {
		port := &service.Spec.Ports[i]
		if port.Name != "" && port.Protocol != "" {
			srvValue := kd.generateSRVRecordValue(service, int(port.Port))

			l := []string{"_" + strings.ToLower(string(port.Protocol)), "_" + port.Name}
			glog.V(2).Infof("Added SRV record %+v", srvValue)

			subCache.SetEntry(recordLabel, srvValue, kd.fqdn(service, append(l, recordLabel)...), l...)
		}
	}
	subCachePath := append(kd.domainPath, serviceSubdomain, service.Namespace)
	host := getServiceFQDN(kd.domain, service)
	reverseRecord, _ := util.GetSkyMsg(host, 0)

	kd.cacheLock.Lock()
	defer kd.cacheLock.Unlock()
	kd.cache.SetSubCache(service.Name, subCache, subCachePath...)
	kd.reverseRecordMap[service.Spec.ClusterIP] = reverseRecord
	kd.clusterIPServiceMap[service.Spec.ClusterIP] = service
}
```

在 KubeDNS 的 Start 函数中启动接受时间的主体循环。
 
```
func (kd *KubeDNS) Start() {
	glog.V(2).Infof("Starting endpointsController")
	go kd.endpointsController.Run(wait.NeverStop)

	glog.V(2).Infof("Starting serviceController")
	go kd.serviceController.Run(wait.NeverStop)

	kd.startConfigMapSync()

	// Wait synchronously for the initial list operations to be
	// complete of endpoints and services from APIServer.
	kd.waitForResourceSyncedOrDie()
}
```

[refer](https://www.kubernetes.org.cn/542.html)
